package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	demoApiaryID = "apiary-north"
	demoHiveID   = "hive-01"
)

func seedApiary(c *client, id, name, district, keeper string) error {
	_, err := c.expect("POST", "/apiaries", map[string]any{
		"id":       id,
		"name":     name,
		"district": district,
		"keeper":   keeper,
	}, 201, `"id":"`+id+`"`)
	return err
}

func seedHive(c *client, apiaryID, hiveID, model string) error {
	_, err := c.expect("POST", "/apiaries/"+apiaryID+"/hives", map[string]any{
		"id":    hiveID,
		"model": model,
	}, 201, `"state":"colonized"`)
	return err
}

func seedDefault(c *client) error {
	if err := seedApiary(c, demoApiaryID, "北岭蜂场", "北岭片区", "周岚"); err != nil {
		return err
	}
	return seedHive(c, demoApiaryID, demoHiveID, "langstroth")
}

func checkRegisterApiary(c *client) error {
	if _, err := c.expect("POST", "/apiaries", map[string]any{
		"id": "apiary-north", "name": "北岭蜂场", "district": "北岭片区", "keeper": "周岚",
	}, 201, `"id":"apiary-north"`, `"name":"北岭蜂场"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north", nil, 200,
		`"district":"北岭片区"`, `"keeper":"周岚"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries", map[string]any{
		"id": "apiary-north", "name": "重复蜂场", "district": "北岭片区", "keeper": "周岚",
	}, 409, `"code":"conflict"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries", map[string]any{
		"id": "apiary-south", "name": "", "district": "南坡片区", "keeper": "李澈",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}

func checkHiveInstall(c *client) error {
	if err := seedApiary(c, demoApiaryID, "北岭蜂场", "北岭片区", "周岚"); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-north/hives", map[string]any{
		"id": "hive-01", "model": "langstroth",
	}, 201, `"state":"colonized"`, `"apiaryId":"apiary-north"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01", nil, 200, `"model":"langstroth"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-north/hives", map[string]any{
		"id": "hive-01", "model": "warre",
	}, 409, `"code":"conflict"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-missing/hives", map[string]any{
		"id": "hive-09", "model": "warre",
	}, 404, `"code":"not_found"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/apiaries/apiary-north/hives", map[string]any{
		"id": "hive-02", "model": "dadant",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}

func checkRecordInspection(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 9, "broodPattern": "patchy",
		"temper": "calm", "miteCount": 3, "note": "",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 3, "broodPattern": "patchy",
		"temper": "aggressive", "miteCount": 12, "note": "西侧巢门观察",
	}, 201, `"inspection-000001"`, `"mite_load"`, `"temper_risk"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01/inspections", nil, 200, `"note":"西侧巢门观察"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-11T09:00:00Z", "colonyStrength": 3, "broodPattern": "solid",
		"temper": "calm", "miteCount": 1, "note": "",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}
	return nil
}

func checkReplaceQueen(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	// 尚未有产卵蜂王时不能登记失王。
	if _, err := c.expect("POST", "/hives/hive-01/queens/loss", map[string]any{}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	// 建立第一只产卵蜂王。
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "bred", "introducedAt": "2026-04-02T08:00:00Z", "reason": "aging",
	}, 201, `"queen-000001"`, `"markColor":"white"`, `"state":"laying"`); err != nil {
		return err
	}
	// 常规换王：旧记录关闭为 superseded，新记录产卵。
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "cells", "introducedAt": "2026-04-20T08:00:00Z", "reason": "aggression",
	}, 201, `"queen-000002"`, `"markColor":"white"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01/queens", nil, 200,
		`"queen-000001"`, `"superseded"`, `"queen-000002"`); err != nil {
		return err
	}
	// 非法来源仍在校验层拒绝。
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "imported", "introducedAt": "2026-05-01T08:00:00Z", "reason": "aging",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	// 失王不能借建立入口提交；必须单独登记。
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "bred", "introducedAt": "2026-05-01T08:00:00Z", "reason": "loss",
	}, 400, `"code":"invalid_request"`, `"reason"`); err != nil {
		return err
	}
	// 登记失王：只关闭原记录为 lost，不新建蜂王。
	if _, err := c.expect("POST", "/hives/hive-01/queens/loss", map[string]any{
		"observedAt": "2026-05-03T08:00:00Z",
	}, 200, `"queen-000002"`, `"lost"`, `"closedAt":"2026-05-03T08:00:00Z"`); err != nil {
		return err
	}
	// 失王后蜂箱不能继续显示为有产卵蜂王。
	if err := assertQueenFact(c, "hive-01", ""); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-05-04T00:00:00Z", nil, 200,
		`"layingQueens":0`, `"hiveId":"hive-01"`, `"queenId":""`); err != nil {
		return err
	}
	// 重复登记失王被拒绝（事实已经是失王）。
	if _, err := c.expect("POST", "/hives/hive-01/queens/loss", map[string]any{}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	// 用新记录重新建立产卵蜂王，旧的 lost 记录保持不变。
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "swarm", "introducedAt": "2027-03-15T08:00:00Z",
	}, 201, `"queen-000003"`, `"markColor":"yellow"`, `"state":"laying"`); err != nil {
		return err
	}
	// 重建后只有唯一一只产卵蜂王，且活动蜂王就是新记录。
	if err := assertQueenFact(c, "hive-01", "queen-000003"); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/hives/hive-01/queens", nil, 200,
		`"queen-000002"`, `"lost"`, `"queen-000003"`, `"laying"`); err != nil {
		return err
	}
	// 并发重复登记失王：至多一次成功，其余全部冲突，最终事实仍唯一。
	if err := assertConcurrentLoss(c, demoApiaryID, "hive-02"); err != nil {
		return err
	}
	// 失王与建立新王并发交错多轮：最终仍只有唯一产卵蜂王。
	if err := assertConcurrentLossAndEstablish(c, demoApiaryID, "hive-03"); err != nil {
		return err
	}
	// 已退役蜂箱既不能登记失王，也不能建立新蜂王。
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens/loss", map[string]any{}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/queens", map[string]any{
		"origin": "bred", "introducedAt": "2027-04-01T08:00:00Z",
	}, 409, `"code":"invalid_state"`); err != nil {
		return err
	}
	// 未知蜂箱返回未找到。
	if _, err := c.expect("POST", "/hives/hive-missing/queens/loss", map[string]any{}, 404,
		`"code":"not_found"`); err != nil {
		return err
	}
	// observedAt 时间格式非法返回校验错误。
	if _, err := c.expect("POST", "/hives/hive-02/queens/loss", map[string]any{
		"observedAt": "not-a-time",
	}, 400, `"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}

// assertQueenFact 断言蜂箱最终蜂王事实：产卵记录恰好为 wantActiveID（空串表示失王），
// 且蜂箱的活动蜂王编号与之一致；失王或已关闭的记录不得残留为 laying。
func assertQueenFact(c *client, hiveID, wantActiveID string) error {
	reply, err := c.call("GET", "/hives/"+hiveID+"/queens", nil)
	if err != nil {
		return err
	}
	if reply.status != 200 {
		return fmt.Errorf("GET /hives/%s/queens 期望 200，实际 %d：%s", hiveID, reply.status, reply.text())
	}
	var records []struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(reply.body, &records); err != nil {
		return fmt.Errorf("蜂王记录无法解析：%v：%s", err, reply.text())
	}
	laying := []string{}
	for _, record := range records {
		if record.State == "laying" {
			laying = append(laying, record.ID)
		}
	}
	if len(laying) > 1 {
		return fmt.Errorf("蜂箱 %s 出现 %d 只产卵蜂王 %v：%s", hiveID, len(laying), laying, reply.text())
	}
	gotActiveID := ""
	if len(laying) == 1 {
		gotActiveID = laying[0]
	}
	if gotActiveID != wantActiveID {
		return fmt.Errorf("蜂箱 %s 产卵蜂王期望 %q，实际 %q：%s", hiveID, wantActiveID, gotActiveID, reply.text())
	}
	hiveReply, err := c.call("GET", "/hives/"+hiveID, nil)
	if err != nil {
		return err
	}
	var hive struct {
		ActiveQueenID string `json:"activeQueenId"`
	}
	if err := json.Unmarshal(hiveReply.body, &hive); err != nil {
		return fmt.Errorf("蜂箱无法解析：%v：%s", err, hiveReply.text())
	}
	if hive.ActiveQueenID != wantActiveID {
		return fmt.Errorf("蜂箱 %s activeQueenId 期望 %q，实际 %q", hiveID, wantActiveID, hive.ActiveQueenID)
	}
	return nil
}

// assertConcurrentLoss 验证并发重复登记失王：只有一次成功，其余冲突，
// 蜂箱最终恰好失王一次且无产卵蜂王。
func assertConcurrentLoss(c *client, apiaryID, hiveID string) error {
	if err := seedHive(c, apiaryID, hiveID, "langstroth"); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/"+hiveID+"/queens", map[string]any{
		"origin": "bred", "introducedAt": "2026-04-02T08:00:00Z",
	}, 201, `"state":"laying"`); err != nil {
		return err
	}
	const attempts = 16
	var wg sync.WaitGroup
	statuses := make([]int, attempts)
	var mu sync.Mutex
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			reply, err := c.call("POST", "/hives/"+hiveID+"/queens/loss", map[string]any{})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				statuses[index] = -1
				return
			}
			statuses[index] = reply.status
		}(i)
	}
	wg.Wait()
	success, conflict := 0, 0
	for index, status := range statuses {
		switch status {
		case 200:
			success++
		case 409:
			conflict++
		default:
			return fmt.Errorf("并发失王第 %d 次返回意外状态 %d", index, status)
		}
	}
	if success != 1 {
		return fmt.Errorf("并发失王成功次数期望 1，实际 %d（冲突 %d）", success, conflict)
	}
	if conflict != attempts-1 {
		return fmt.Errorf("并发失王冲突次数期望 %d，实际 %d", attempts-1, conflict)
	}
	return assertQueenFact(c, hiveID, "")
}

// assertConcurrentLossAndEstablish 验证失王与建立新王并发交错多轮后，
// 蜂箱的最终事实唯一：恰好一只产卵蜂王，且活动蜂王指向同一条记录。
func assertConcurrentLossAndEstablish(c *client, apiaryID, hiveID string) error {
	if err := seedHive(c, apiaryID, hiveID, "langstroth"); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/"+hiveID+"/queens", map[string]any{
		"origin": "bred", "introducedAt": "2026-04-02T08:00:00Z",
	}, 201, `"state":"laying"`); err != nil {
		return err
	}
	var wg sync.WaitGroup
	for round := 0; round < 4; round++ {
		introducedAt := time.Date(2027, time.Month(1+round), 10, 8, 0, 0, 0, time.UTC).Format(time.RFC3339)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = c.call("POST", "/hives/"+hiveID+"/queens/loss", map[string]any{})
		}()
		go func() {
			defer wg.Done()
			_, _ = c.call("POST", "/hives/"+hiveID+"/queens", map[string]any{
				"origin":       "swarm",
				"introducedAt": introducedAt,
			})
		}()
		wg.Wait()
	}
	reply, err := c.call("GET", "/hives/"+hiveID+"/queens", nil)
	if err != nil {
		return err
	}
	var records []struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(reply.body, &records); err != nil {
		return fmt.Errorf("蜂王记录无法解析：%v：%s", err, reply.text())
	}
	laying := []string{}
	for _, record := range records {
		if record.State != "laying" && record.State != "superseded" && record.State != "lost" {
			return fmt.Errorf("蜂王 %s 出现未知状态 %q", record.ID, record.State)
		}
		if record.State == "laying" {
			laying = append(laying, record.ID)
		}
	}
	if len(laying) > 1 {
		return fmt.Errorf("并发交错后蜂箱 %s 出现 %d 只产卵蜂王 %v", hiveID, len(laying), laying)
	}
	if len(laying) == 1 {
		return assertQueenFact(c, hiveID, laying[0])
	}
	return assertQueenFact(c, hiveID, "")
}

func checkAdvanceHiveState(c *client) error {
	if err := seedDefault(c); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "production"}, 200,
		`"from":"colonized"`, `"to":"production"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "colonized"}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "production"}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "retired"}, 200,
		`"to":"retired"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/state", map[string]any{"target": "production"}, 409,
		`"code":"invalid_state"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-missing/state", map[string]any{"target": "production"}, 404,
		`"code":"not_found"`); err != nil {
		return err
	}
	return nil
}

func checkReadApiaryStatus(c *client) error {
	if err := seedApiary(c, demoApiaryID, "北岭蜂场", "北岭片区", "周岚"); err != nil {
		return err
	}
	for _, hiveID := range []string{"hive-01", "hive-02", "hive-03"} {
		if err := seedHive(c, demoApiaryID, hiveID, "langstroth"); err != nil {
			return err
		}
	}
	if _, err := c.expect("POST", "/hives/hive-02/state", map[string]any{"target": "production"}, 200,
		`"to":"production"`); err != nil {
		return err
	}
	if _, err := c.expect("POST", "/hives/hive-01/inspections", map[string]any{
		"observedAt": "2026-04-10T09:00:00Z", "colonyStrength": 4, "broodPattern": "solid",
		"temper": "calm", "miteCount": 12, "note": "",
	}, 201, `"mite_load"`); err != nil {
		return err
	}
	report, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=2026-04-12T00:00:00Z", nil, 200,
		`"hiveCount":3`, `"colonized":2`, `"production":1`, `"layingQueens":0`)
	if err != nil {
		return err
	}
	for _, fragment := range []string{
		`"needsInspection":["hive-02","hive-03"]`,
		`"concernHives":["hive-01"]`,
	} {
		if !strings.Contains(report.text(), fragment) {
			return fmt.Errorf("状态汇总缺少 %q：%s", fragment, report.text())
		}
	}
	if err := report.mustOrder(`"hiveId":"hive-01"`, `"hiveId":"hive-02"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-missing/status", nil, 404, `"code":"not_found"`); err != nil {
		return err
	}
	if _, err := c.expect("GET", "/apiaries/apiary-north/status?asOf=not-a-time", nil, 400,
		`"code":"invalid_request"`); err != nil {
		return err
	}
	return nil
}
