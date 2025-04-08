package plex

import (
	"context"
	"log"

	common "github.com/aquare11e/media-dowloader-bot/common/protogen/common"
	protoPlex "github.com/aquare11e/media-dowloader-bot/common/protogen/plex"
)

type PlexService struct {
	protoPlex.UnimplementedPlexServiceServer
	client *Client
}

func NewPlexService(baseURL string, token string, pbTypeToCategoryId map[common.RequestType]string) *PlexService {
	client := NewClient(baseURL, token, pbTypeToCategoryId)
	return &PlexService{
		client: client,
	}
}

func (s *PlexService) UpdateCategory(ctx context.Context, in *protoPlex.UpdateCategoryRequest) (*protoPlex.UpdateCategoryResponse, error) {
	log.Println("UpdateCategory")

	// Call the ScanLibrary method or any other relevant method to update the category
	err := s.client.ScanLibrary(ctx, in.Type)
	if err != nil {
		log.Printf("Failed to update category: %v", err)
		return nil, err
	}

	// Return a successful response
	return &protoPlex.UpdateCategoryResponse{
		RequestId: in.RequestId,
		Result:    protoPlex.ResponseResult_RESPONSE_RESULT_SUCCESS,
		Message:   "Category updated successfully",
	}, nil
}
