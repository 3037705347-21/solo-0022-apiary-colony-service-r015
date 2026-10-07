package queen

import (
	"time"

	"example.com/apiary-colony-service/internal/domain"
	"example.com/apiary-colony-service/internal/store"
)

const (
	ReasonAging      = "aging"
	ReasonAggression = "aggression"
)

// Service 处理蜂王生命周期：登记失王（只关闭旧记录）与建立/更换产卵蜂王。
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

type EstablishInput struct {
	Origin       string
	IntroducedAt time.Time
	Reason       string
}

type LossInput struct {
	ObservedAt time.Time
}

// Establish 建立或更换蜂箱的产卵蜂王：存在产卵蜂王时先关闭为 superseded，
// 失王期间调用则直接重新建立。失王事实只能通过 ReportLoss 单独登记。
func (s *Service) Establish(hiveID string, input EstablishInput) (domain.Queen, error) {
	if _, ok := s.store.GetHive(hiveID); !ok {
		return domain.Queen{}, domain.NotFound("蜂箱", hiveID)
	}
	origin, err := ParseOrigin(input.Origin)
	if err != nil {
		return domain.Queen{}, err
	}
	if domain.TrimText(input.Reason) != "" {
		if _, err := ParseReason(input.Reason); err != nil {
			return domain.Queen{}, err
		}
	}
	introducedAt := input.IntroducedAt
	if introducedAt.IsZero() {
		introducedAt = s.clock()
	}
	record, err := domain.NewQueen(hiveID, introducedAt, origin, ColorForIntroduction(introducedAt))
	if err != nil {
		return domain.Queen{}, err
	}
	return s.store.EstablishQueen(hiveID, record, s.clock())
}

// ReportLoss 仅登记失王事实：关闭当前产卵蜂王记录但不新建蜂王。
func (s *Service) ReportLoss(hiveID string, input LossInput) (domain.Queen, error) {
	if _, ok := s.store.GetHive(hiveID); !ok {
		return domain.Queen{}, domain.NotFound("蜂箱", hiveID)
	}
	observedAt := input.ObservedAt
	if observedAt.IsZero() {
		observedAt = s.clock()
	}
	return s.store.ReportQueenLoss(hiveID, observedAt)
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

// ParseReason 校验建立/更换产卵王的原因。失王不是建立原因：
// 失王事实必须通过单独的失王登记入口提交。
func ParseReason(value string) (string, error) {
	reason := domain.TrimText(value)
	switch reason {
	case ReasonAging, ReasonAggression:
		return reason, nil
	default:
		return "", domain.Invalid(
			"reason",
			"reason 必须是 %s 或 %s；确认失王请走失王登记入口",
			ReasonAging, ReasonAggression,
		)
	}
}
