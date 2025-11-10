package service

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	blog "github.com/yourEmotion/Blog_gRPC/api/go"
	"github.com/yourEmotion/Blog_gRPC/internal/models"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

// Метрика времени Redis для лайков
// Для анализа времени ответа Redis выбрал Histogram
// Так удобнее отыскивать среднее, медиану и отлавливать хвосты (медленные запросы)
var (
	redisLikesDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "redis_likes_duration_seconds",
		Help:    "Time taken to fetch likes from Redis",
		Buckets: prometheus.DefBuckets,
	})
)

func init() {
	prometheus.MustRegister(redisLikesDuration)
}

type BlogService struct {
	blog.UnimplementedBlogServiceServer
	db          *gorm.DB
	redisClient *redis.Client
}

func NewBlogService(db *gorm.DB, redisClient *redis.Client) *BlogService {
	return &BlogService{
		db:          db,
		redisClient: redisClient,
	}
}

func userIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Println("No metadata in context")
		return ""
	}
	vals := md.Get("User-Id")
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func execRedisWithLogging(ctx context.Context, pipe redis.Pipeliner, operation string) error {
	start := time.Now()
	log.Printf("[REDIS START] %s", operation)
	_, err := pipe.Exec(ctx)
	duration := time.Since(start).Seconds()
	if err != nil {
		log.Printf("[REDIS ERROR] %s duration=%.3fs err=%v", operation, duration, err)
	} else {
		log.Printf("[REDIS END] %s duration=%.3fs", operation, duration)
	}

	redisLikesDuration.Observe(duration)
	return err
}

func (s *BlogService) GetPosts(ctx context.Context, req *blog.GetPostsRequest) (*blog.GetPostsResponse, error) {
	var postsDB []models.Post
	cacheKey := "main:feed"
	var hasCache = false
	if req.Limit == 10 && req.Offset == 0 {
		cached, err := s.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var cachedPosts []models.Post
			if err := json.Unmarshal([]byte(cached), &cachedPosts); err == nil {
				postsDB = cachedPosts
				hasCache = true
			}
		}
	}

	if !hasCache {
		if err := s.db.Order("created_at desc").
			Limit(int(req.Limit)).
			Offset(int(req.Offset)).
			Find(&postsDB).Error; err != nil {
			return nil, err
		}

		if req.Limit == 10 && req.Offset == 0 {
			data, _ := json.Marshal(postsDB)
			s.redisClient.Set(ctx, cacheKey, data, 30*time.Second)
		}
	}

	userID := userIDFromContext(ctx)
	posts := make([]*blog.Post, len(postsDB))

	pipe := s.redisClient.Pipeline()
	likeCounts := make([]*redis.IntCmd, len(postsDB))
	likedByUser := make([]*redis.BoolCmd, len(postsDB))
	for i, p := range postsDB {
		key := "post:likes:" + strconv.FormatUint(p.ID, 10)
		likeCounts[i] = pipe.SCard(ctx, key)
		if userID != "" {
			likedByUser[i] = pipe.SIsMember(ctx, key, userID)
		}
	}

	execRedisWithLogging(ctx, pipe, "Get likes")

	for i, p := range postsDB {
		liked := false
		if likedByUser[i] != nil {
			liked = likedByUser[i].Val()
		}
		posts[i] = &blog.Post{
			Id:          p.ID,
			AuthorId:    p.AuthorID,
			CreatedAt:   p.CreatedAt.Format(time.RFC3339),
			Body:        p.Body,
			LikesCount:  uint64(likeCounts[i].Val()),
			LikedByUser: liked,
		}
	}

	return &blog.GetPostsResponse{Posts: posts}, nil
}

func (s *BlogService) CreatePost(ctx context.Context, req *blog.CreatePostRequest) (*blog.Post, error) {
	userID := userIDFromContext(ctx)
	var uid uint64 = 0
	if userID != "" {
		parsed, _ := strconv.ParseUint(userID, 10, 64)
		uid = parsed
	}

	post := models.Post{
		AuthorID:  uid,
		Body:      req.Body,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(&post).Error; err != nil {
		return nil, err
	}

	return &blog.Post{
		Id:         post.ID,
		AuthorId:   post.AuthorID,
		CreatedAt:  post.CreatedAt.Format(time.RFC3339),
		Body:       post.Body,
		LikesCount: 0,
	}, nil
}

func (s *BlogService) EditPost(ctx context.Context, req *blog.EditPostRequest) (*blog.Post, error) {
	var post models.Post
	if err := s.db.First(&post, req.PostId).Error; err != nil {
		return nil, err
	}

	post.Body = req.Body
	if err := s.db.Save(&post).Error; err != nil {
		return nil, err
	}

	userID := userIDFromContext(ctx)
	key := "post:likes:" + strconv.FormatUint(post.ID, 10)
	likes, _ := s.redisClient.SCard(ctx, key).Result()
	liked := false
	if userID != "" {
		liked, _ = s.redisClient.SIsMember(ctx, key, userID).Result()
	}

	return &blog.Post{
		Id:          post.ID,
		AuthorId:    post.AuthorID,
		CreatedAt:   post.CreatedAt.Format(time.RFC3339),
		Body:        post.Body,
		LikesCount:  uint64(likes),
		LikedByUser: liked,
	}, nil
}

func (s *BlogService) DeletePost(ctx context.Context, req *blog.DeletePostRequest) (*blog.EmptyResponse, error) {
	if err := s.db.Delete(&models.Post{}, req.PostId).Error; err != nil {
		return nil, err
	}
	s.redisClient.Del(ctx, "post:likes:"+strconv.FormatInt(req.PostId, 10))
	return &blog.EmptyResponse{}, nil
}

func (s *BlogService) LikePost(ctx context.Context, req *blog.LikePostRequest) (*blog.EmptyResponse, error) {
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, nil
	}
	key := "post:likes:" + strconv.FormatInt(req.PostId, 10)
	s.redisClient.SAdd(ctx, key, userID)
	return &blog.EmptyResponse{}, nil
}

func (s *BlogService) UnlikePost(ctx context.Context, req *blog.UnlikePostRequest) (*blog.EmptyResponse, error) {
	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, nil
	}
	key := "post:likes:" + strconv.FormatInt(req.PostId, 10)
	s.redisClient.SRem(ctx, key, userID)
	return &blog.EmptyResponse{}, nil
}
