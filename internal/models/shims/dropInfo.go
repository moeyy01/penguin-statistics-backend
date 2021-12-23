package shims

import (
	"encoding/json"

	"github.com/uptrace/bun"
)

type DropInfo struct {
	bun.BaseModel `bun:"drop_infos"`

	DropID     int64           `bun:",pk" json:"dropId"`
	Server     string          `json:"-"`
	StageID    int64           `json:"-"`
	ItemID     int64           `json:"-"`
	ArkStageID string          `bun:"-" json:"stageId"`
	ArkItemID  string          `bun:"-" json:"itemId"`
	DropType   string          `json:"dropType"`
	Bounds     json.RawMessage `json:"bounds"`

	Item  *Item  `bun:"rel:belongs-to,join:item_id=item_id" json:"-"`
	Stage *Stage `bun:"rel:belongs-to,join:stage_id=stage_id" json:"-"`
}