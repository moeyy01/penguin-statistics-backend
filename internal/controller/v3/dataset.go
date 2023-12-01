package v3

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jinzhu/copier"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"gopkg.in/guregu/null.v3"

	modelv2 "exusiai.dev/backend-next/internal/model/v2"
	modelv3 "exusiai.dev/backend-next/internal/model/v3"
	"exusiai.dev/backend-next/internal/server/svr"
	"exusiai.dev/backend-next/internal/service"
	"exusiai.dev/backend-next/internal/util/rekuest"
)

type Dataset struct {
	fx.In

	AccountService       *service.Account
	DropMatrixService    *service.DropMatrix
	TrendService         *service.Trend
	PatternMatrixService *service.PatternMatrix
}

func RegisterDataset(v3 *svr.V3, c Dataset) {
	dataset := v3.Group("/dataset")
	aggregated := dataset.Group("/aggregated/:source/:category/:server")
	aggregated.Get("/item/:itemId", c.AggregatedItem)
	aggregated.Get("/stage/:stageId", c.AggregatedStage)
}

func (c Dataset) aggregateMatrix(ctx *fiber.Ctx) (*modelv2.DropMatrixQueryResult, error) {
	server := ctx.Params("server", "CN")
	if err := rekuest.ValidServer(ctx, server); err != nil {
		return nil, err
	}

	category := ctx.Params("category", "all")
	if err := rekuest.ValidCategory(ctx, category); err != nil {
		return nil, err
	}

	isPersonal := ctx.Params("source") == "personal"

	accountId := null.NewInt(0, false)
	if isPersonal {
		account, err := c.AccountService.GetAccountFromRequest(ctx)
		if err != nil {
			return nil, err
		}
		accountId.Int64 = int64(account.AccountID)
		accountId.Valid = true
	}

	return c.DropMatrixService.GetShimDropMatrix(ctx.UserContext(), server, true, "", "", accountId, category)
}

func (c Dataset) aggregateTrend(ctx *fiber.Ctx) (*modelv2.TrendQueryResult, error) {
	server := ctx.Params("server", "CN")
	if err := rekuest.ValidServer(ctx, server); err != nil {
		return nil, err
	}

	result, err := c.TrendService.GetShimTrend(ctx.UserContext(), server)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c Dataset) aggregatePattern(ctx *fiber.Ctx) (*modelv3.PatternMatrixQueryResult, error) {
	server := ctx.Query("server", "CN")
	if err := rekuest.ValidServer(ctx, server); err != nil {
		return nil, err
	}

	showAllPatterns := ctx.Query("show_all_patterns", "false") == "true"

	isPersonal, err := strconv.ParseBool(ctx.Query("is_personal", "false"))
	if err != nil {
		return nil, err
	}

	accountId := null.NewInt(0, false)
	if isPersonal {
		account, err := c.AccountService.GetAccountFromRequest(ctx)
		if err != nil {
			return nil, err
		}
		accountId.Int64 = int64(account.AccountID)
		accountId.Valid = true
	}

	shimResult, err := c.PatternMatrixService.GetShimPatternMatrix(ctx.UserContext(), server, accountId, ctx.Params("category"), showAllPatterns)
	if err != nil {
		return nil, err
	}

	var result modelv3.PatternMatrixQueryResult
	copier.Copy(&result, shimResult)

	return &result, nil
}

func (c Dataset) AggregatedItem(ctx *fiber.Ctx) error {
	aggregated := &modelv3.AggregatedItemStats{}
	itemId := ctx.Params("itemId")

	matrix, err := c.aggregateMatrix(ctx)
	if err != nil {
		return err
	}
	aggregated.Matrix = lo.Filter(matrix.Matrix, func(el *modelv2.OneDropMatrixElement, _ int) bool {
		return el.ItemID == itemId
	})

	trend, err := c.aggregateTrend(ctx)
	if err != nil {
		return err
	}
	aggregated.Trends = make(map[string]*modelv2.StageTrend)
	for stageId, v := range trend.Trend {
		for itemId, vv := range v.Results {
			if itemId != ctx.Params("itemId") {
				continue
			}
			if _, ok := aggregated.Trends[stageId]; !ok {
				aggregated.Trends[stageId] = &modelv2.StageTrend{
					StartTime: v.StartTime,
					Results:   make(map[string]*modelv2.OneItemTrend),
				}
			}
			aggregated.Trends[stageId].Results[itemId] = vv
		}
	}

	return ctx.JSON(aggregated)
}

func (c Dataset) AggregatedStage(ctx *fiber.Ctx) error {
	aggregated := &modelv3.AggregatedStageStats{}
	stageId := ctx.Params("stageId")

	matrix, err := c.aggregateMatrix(ctx)
	if err != nil {
		return err
	}
	aggregated.Matrix = lo.Filter(matrix.Matrix, func(el *modelv2.OneDropMatrixElement, _ int) bool {
		return el.StageID == stageId
	})

	trend, err := c.aggregateTrend(ctx)
	if err != nil {
		return err
	}
	aggregated.Trends = make(map[string]*modelv2.StageTrend)
	for trendStageId, v := range trend.Trend {
		if trendStageId != stageId {
			continue
		}
		for itemId, vv := range v.Results {
			if _, ok := aggregated.Trends[trendStageId]; !ok {
				aggregated.Trends[trendStageId] = &modelv2.StageTrend{
					StartTime: v.StartTime,
					Results:   make(map[string]*modelv2.OneItemTrend),
				}
			}
			aggregated.Trends[trendStageId].Results[itemId] = vv
		}
	}

	pattern, err := c.aggregatePattern(ctx)
	if err != nil {
		return err
	}
	aggregated.Patterns = lo.Filter(pattern.PatternMatrix, func(el *modelv3.OnePatternMatrixElement, _ int) bool {
		return el.StageID == stageId
	})

	return ctx.JSON(aggregated)
}
