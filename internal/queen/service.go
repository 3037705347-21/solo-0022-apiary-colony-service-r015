package queen

import (
	"time"

	"example.com/apiary-colony-service/internal/domain"
	"example.com/apiary-colony-service/internal/store"
)

const (
	ReasonAging      = "aging"
	ReasonAggression = "aggression"
	ReasonLoss       = "loss"
)

// Service 处理蜂王生命周期：登记失王、只建立新产卵蜂王，或一次完成换王。
type Service struct {
	store *store.Memory
	clock func() time.Time
}

func New(memory *store.Memory, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{store: memory, clock: clock}
}

type ReplaceInput struct {
	Origin       string
	IntroducedAt time.Time
	Reason       string
}

type LossInput struct {
	ObservedAt time.Time
}

type EstablishInput struct {
	Origin       string
	IntroducedAt time.Time
}

// ReportLoss 只登记失王事实：关闭当前产卵记录，并让蜂箱暂时没有当前蜂王。
func (s *Service) ReportLoss(hiveID string, input LossInput) (domain.Queen, error) {
	hive, ok := s.store.GetHive(hiveID)
	if !ok {
		return domain.Queen{}, domain.NotFound("蜂箱", hiveID)
	}
	observedAt := input.ObservedAt
	if observedAt.IsZero() {
		observedAt = s.clock()
	}
	return s.store.CloseActiveQueen(hiveID, hive.ActiveQueenID, domain.QueenLost, observedAt)
}

// Establish 在失王之后建立新的产卵蜂王；不会关闭旧记录。
func (s *Service) Establish(hiveID string, input EstablishInput) (domain.Queen, error) {
	origin, err := ParseOrigin(input.Origin)
	if err != nil {
		return domain.Queen{}, err
	}
	introducedAt := input.IntroducedAt
	if introducedAt.IsZero() {
		introducedAt = s.clock()
	}
	record, err := domain.NewQueen(hiveID, introducedAt, origin, ColorForIntroduction(introducedAt))
	if err != nil {
		return domain.Queen{}, err
	}
	return s.store.AddQueen(hiveID, record, "", "", s.clock())
}

func (s *Service) Replace(hiveID string, input ReplaceInput) (domain.Queen, error) {
	hive, ok := s.store.GetHive(hiveID)
	if !ok {
		return domain.Queen{}, domain.NotFound("蜂箱", hiveID)
	}
	origin, err := ParseOrigin(input.Origin)
	if err != nil {
		return domain.Queen{}, err
	}
	introducedAt := input.IntroducedAt
	if introducedAt.IsZero() {
		introducedAt = s.clock()
	}
	record, err := domain.NewQueen(hiveID, introducedAt, origin, ColorForIntroduction(introducedAt))
	if err != nil {
		return domain.Queen{}, err
	}
	if hive.ActiveQueenID == "" {
		return s.store.AddQueen(hiveID, record, "", "", s.clock())
	}
	reason, err := ParseReason(input.Reason)
	if err != nil {
		return domain.Queen{}, err
	}
	closeState := domain.QueenSuperseded
	if reason == ReasonLoss {
		closeState = domain.QueenLost
	}
	return s.store.AddQueen(hiveID, record, hive.ActiveQueenID, closeState, s.clock())
}

func (s *Service) List(hiveID string) ([]domain.Queen, error) {
	return s.store.ListQueens(hiveID)
}

func ParseOrigin(value string) (domain.QueenOrigin, error) {
	origin := domain.QueenOrigin(domain.TrimText(value))
	if !domain.ValidQueenOrigin(origin) {
		return "", domain.Invalid(
			"origin",
			"origin 必须是 %s、%s 或 %s",
			domain.OriginBred, domain.OriginCells, domain.OriginSwarm,
		)
	}
	return origin, nil
}

func ParseReason(value string) (string, error) {
	reason := domain.TrimText(value)
	switch reason {
	case ReasonAging, ReasonAggression, ReasonLoss:
		return reason, nil
	default:
		return "", domain.Invalid("reason", "reason 必须是 %s、%s 或 %s", ReasonAging, ReasonAggression, ReasonLoss)
	}
}
