package httpapi

import (
	"net/http"

	"example.com/apiary-colony-service/internal/domain"
	"example.com/apiary-colony-service/internal/queen"
)

type establishQueenRequest struct {
	Origin       string `json:"origin"`
	IntroducedAt string `json:"introducedAt"`
	Reason       string `json:"reason"`
}

type reportQueenLossRequest struct {
	ObservedAt string `json:"observedAt"`
}

// handleEstablishQueen 建立或更换产卵蜂王：失王期间调用即重新建立产卵蜂王。
func (s *Server) handleEstablishQueen(w http.ResponseWriter, r *http.Request) {
	var request establishQueenRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}
	introducedAt, err := parseOptionalTime("introducedAt", request.IntroducedAt)
	if err != nil {
		writeError(w, err)
		return
	}
	input := queen.EstablishInput{
		Origin: request.Origin,
		Reason: request.Reason,
	}
	if introducedAt != nil {
		input.IntroducedAt = *introducedAt
	}
	record, err := s.deps.Queen.Establish(r.PathValue("hiveId"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, record)
}

// handleReportQueenLoss 只登记失王事实：关闭原产卵记录，不新建蜂王。
// 请求体允许为空对象，observedAt 缺省取服务当前时间。
func (s *Server) handleReportQueenLoss(w http.ResponseWriter, r *http.Request) {
	var request reportQueenLossRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(w, r, &request); err != nil {
			writeError(w, err)
			return
		}
	}
	observedAt, err := parseOptionalTime("observedAt", request.ObservedAt)
	if err != nil {
		writeError(w, err)
		return
	}
	input := queen.LossInput{}
	if observedAt != nil {
		input.ObservedAt = *observedAt
	}
	record, err := s.deps.Queen.ReportLoss(r.PathValue("hiveId"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleListQueens(w http.ResponseWriter, r *http.Request) {
	items, err := s.deps.Queen.List(r.PathValue("hiveId"))
	if err != nil {
		writeError(w, err)
		return
	}
	if items == nil {
		items = []domain.Queen{}
	}
	writeJSON(w, http.StatusOK, items)
}
