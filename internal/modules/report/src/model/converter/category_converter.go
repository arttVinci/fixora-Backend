package converter

import (
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/entity"
	"github.com/arttVinci/fixora-Backend/internal/modules/report/src/model"
)

func CategoryToResponse(category *entity.Category) *model.CategoryResponse {
	if category == nil {
		return nil
	}
	return &model.CategoryResponse{
		ID:   category.ID,
		Name: category.Name,
		Slug: category.Slug,
	}
}
