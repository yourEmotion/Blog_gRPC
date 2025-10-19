package main

import (
    "context"
    "time"
	blog "github.com/yourEmotion/Blog_gRPC/api/go"
)

type BlogServer struct {
    blog.UnimplementedBlogServiceServer
}

func (s *BlogServer) GetPosts(ctx context.Context, req *blog.GetPostsRequest) (*blog.GetPostsResponse, error) {
    posts := []*blog.Post{
        {
            Id:          1,
            AuthorId:    101,
            CreatedAt:   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
            Body:        "Сенсация! Карты снова не провязали отзывы!!!",
            LikesCount:  3,
            LikedByUser: true,
        },
        {
            Id:          2,
            AuthorId:    102,
            CreatedAt:   time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
            Body:        "nikhovas стал первым в истории сотрудником Яндекса, кто получил 4 плюса 10 раз подряд!!!",
            LikesCount:  5,
            LikedByUser: false,
        },
    }
    return &blog.GetPostsResponse{Posts: posts}, nil
}

func (s *BlogServer) CreatePost(ctx context.Context, req *blog.CreatePostRequest) (*blog.Post, error) {
    post := &blog.Post{
        Id:         3,
        AuthorId:   101,
        CreatedAt:  time.Now().Format(time.RFC3339),
        Body:       req.Body,
        LikesCount: 0,
    }
    return post, nil
}

func (s *BlogServer) EditPost(ctx context.Context, req *blog.EditPostRequest) (*blog.Post, error) {
    post := &blog.Post{
        Id:         uint64(req.PostId),
        AuthorId:   101,
        CreatedAt:  time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
        Body:       req.Body,
        LikesCount: 2,
    }
    return post, nil
}

func (s *BlogServer) DeletePost(ctx context.Context, req *blog.DeletePostRequest) (*blog.EmptyResponse, error) {
    return &blog.EmptyResponse{}, nil
}

func (s *BlogServer) LikePost(ctx context.Context, req *blog.LikePostRequest) (*blog.EmptyResponse, error) {
    return &blog.EmptyResponse{}, nil
}

func (s *BlogServer) UnlikePost(ctx context.Context, req *blog.UnlikePostRequest) (*blog.EmptyResponse, error) {
    return &blog.EmptyResponse{}, nil
}
