package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const deliveryInitiativeKey = "AIPDC"

func deliveryInitiativeKeyFor(copilot DeliveryCopilot) string {
	prdID := strings.TrimSpace(copilot.PRD.Info.PRDID)
	if prdID == "" || prdID == "AIPDC-PRD-001" {
		return deliveryInitiativeKey
	}
	return deliveryInitiativeKey + "-" + strings.ToUpper(shortID(prdID))
}

type DeliveryApproval struct {
	Role      string `json:"role"`
	Reviewer  string `json:"reviewer"`
	Status    string `json:"status"`
	Comment   string `json:"comment"`
	DecidedAt string `json:"decidedAt,omitempty"`
}

type DeliveryWorkflowState struct {
	Persisted        bool               `json:"persisted"`
	InitiativeID     string             `json:"initiativeId,omitempty"`
	InitiativeKey    string             `json:"initiativeKey,omitempty"`
	VersionID        string             `json:"versionId,omitempty"`
	Version          string             `json:"version,omitempty"`
	Status           string             `json:"status,omitempty"`
	Requirements     int                `json:"requirements"`
	EvidenceLinks    int                `json:"evidenceLinks"`
	ReviewCount      int                `json:"reviewCount"`
	ReadinessScore   int                `json:"readinessScore"`
	ReadinessVerdict string             `json:"readinessVerdict,omitempty"`
	Approvals        []DeliveryApproval `json:"approvals"`
	PendingApprovals int                `json:"pendingApprovals"`
	UpdatedAt        string             `json:"updatedAt,omitempty"`
}

// PersistDeliveryDraft stores the default history-driven delivery artifact as an editable local draft.
func (s *KnowledgeStore) PersistDeliveryDraft() (DeliveryWorkflowState, error) {
	copilot, err := s.BuildDeliveryCopilot()
	if err != nil {
		return DeliveryWorkflowState{}, err
	}
	return s.PersistDeliveryCopilot(copilot)
}

// PersistDeliveryCopilot saves the exact generated PRD snapshot. A research-specific
// PRD receives an independent initiative key, so a new research question cannot overwrite another draft.
func (s *KnowledgeStore) PersistDeliveryCopilot(copilot DeliveryCopilot) (DeliveryWorkflowState, error) {
	snapshot, err := json.Marshal(copilot)
	if err != nil {
		return DeliveryWorkflowState{}, fmt.Errorf("序列化 PRD 草案失败：%w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	initiativeKey := deliveryInitiativeKeyFor(copilot)
	initiativeID := "initiative-" + strings.ToLower(initiativeKey)

	tx, err := s.db.Begin()
	if err != nil {
		return DeliveryWorkflowState{}, err
	}
	rollback := func(cause error) (DeliveryWorkflowState, error) {
		_ = tx.Rollback()
		return DeliveryWorkflowState{}, cause
	}
	if _, err = tx.Exec(`INSERT INTO product_initiatives (id,initiative_key,title,status,owner,current_version,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(initiative_key) DO UPDATE SET title=excluded.title, updated_at=excluded.updated_at`,
		initiativeID, initiativeKey, copilot.PRD.Info.Feature, "draft", copilot.PRD.Info.Owner, "", now, now); err != nil {
		return rollback(fmt.Errorf("保存产品事项失败：%w", err))
	}

	var versionID, version string
	err = tx.QueryRow(`SELECT id,version FROM prd_versions WHERE initiative_id=? AND status='draft' ORDER BY updated_at DESC LIMIT 1`, initiativeID).Scan(&versionID, &version)
	if err != nil && err != sql.ErrNoRows {
		return rollback(fmt.Errorf("读取 PRD 草案版本失败：%w", err))
	}
	if err == sql.ErrNoRows {
		var completed int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM prd_versions WHERE initiative_id=?`, initiativeID).Scan(&completed); err != nil {
			return rollback(fmt.Errorf("计算 PRD 版本失败：%w", err))
		}
		version = fmt.Sprintf("0.%d", completed+1)
		versionID = "prd-version-" + shortID(initiativeID+":"+version)
		if _, err := tx.Exec(`INSERT INTO prd_versions (id,initiative_id,version,status,snapshot_json,change_summary,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
			versionID, initiativeID, version, "draft", string(snapshot), "由 Delivery Copilot 生成的本地草案", now, now); err != nil {
			return rollback(fmt.Errorf("创建 PRD 版本失败：%w", err))
		}
	} else if _, err := tx.Exec(`UPDATE prd_versions SET snapshot_json=?,change_summary=?,updated_at=? WHERE id=?`, string(snapshot), "重新生成本地草案", now, versionID); err != nil {
		return rollback(fmt.Errorf("更新 PRD 草案失败：%w", err))
	}
	if _, err := tx.Exec(`UPDATE product_initiatives SET current_version=?,status='draft',updated_at=? WHERE id=?`, version, now, initiativeID); err != nil {
		return rollback(fmt.Errorf("更新产品事项版本失败：%w", err))
	}

	// Requirements and their evidence links are rebuilt atomically only while the version is a draft.
	if _, err := tx.Exec(`DELETE FROM prd_requirements WHERE prd_version_id=?`, versionID); err != nil {
		return rollback(fmt.Errorf("刷新草案需求失败：%w", err))
	}
	for _, requirement := range copilot.PRD.Requirements {
		requirementID := "prd-requirement-" + shortID(versionID+":"+requirement.ID)
		preconditions, _ := json.Marshal(requirement.Preconditions)
		mainFlow, _ := json.Marshal(requirement.MainFlow)
		businessRules, _ := json.Marshal(requirement.BusinessRules)
		exceptions, _ := json.Marshal(requirement.Exceptions)
		permissions, _ := json.Marshal(requirement.Permissions)
		acceptance, _ := json.Marshal(requirement.Acceptance)
		tracking, _ := json.Marshal(requirement.Tracking)
		if _, err := tx.Exec(`INSERT INTO prd_requirements (id,prd_version_id,requirement_key,title,priority,user_story,preconditions_json,main_flow_json,business_rules_json,exceptions_json,permissions_json,acceptance_json,tracking_json,status,created_at,updated_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, requirementID, versionID, requirement.ID, requirement.Title, requirement.Priority, requirement.UserStory,
			string(preconditions), string(mainFlow), string(businessRules), string(exceptions), string(permissions), string(acceptance), string(tracking), "draft", now, now); err != nil {
			return rollback(fmt.Errorf("保存需求 %s 失败：%w", requirement.ID, err))
		}
		for index, evidence := range requirement.Evidence {
			sourceType := strings.TrimSpace(evidence.Type)
			if sourceType == "" {
				sourceType = "evidence"
			}
			relation := "supports"
			if sourceType == "gap" {
				relation = "gap"
			}
			sourceID := strings.TrimSpace(evidence.ID)
			if sourceID == "" {
				sourceID = shortID(fmt.Sprintf("%s:%s:%d", requirementID, evidence.Label, index))
			}
			linkID := "evidence-link-" + shortID(requirementID+":"+sourceType+":"+sourceID+":"+relation)
			if _, err := tx.Exec(`INSERT INTO evidence_links (id,prd_requirement_id,source_type,source_id,relation,note,created_at) VALUES (?,?,?,?,?,?,?)`,
				linkID, requirementID, sourceType, sourceID, relation, evidence.Label, now); err != nil {
				return rollback(fmt.Errorf("保存需求证据链接失败：%w", err))
			}
		}
	}

	verdict := copilot.Readiness.Verdict
	reviewStatus := "ready_for_review"
	if copilot.Readiness.Score < 80 {
		reviewStatus = "not_ready"
	}
	reviewChecks, _ := json.Marshal(copilot.Readiness.Checks)
	reviewID := "prd-review-" + shortID(versionID+":"+now)
	if _, err := tx.Exec(`INSERT INTO prd_reviews (id,prd_version_id,reviewer_type,reviewer_name,readiness_score,verdict,status,checks_json,created_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		reviewID, versionID, "rule_based_readiness", "Local Reviewer", copilot.Readiness.Score, verdict, reviewStatus, string(reviewChecks), now); err != nil {
		return rollback(fmt.Errorf("保存 PRD Readiness 评审失败：%w", err))
	}
	for _, role := range []string{"product_owner", "design_lead", "engineering_lead", "qa_lead", "security_lead"} {
		approvalID := "prd-approval-" + shortID(versionID+":"+role)
		if _, err := tx.Exec(`INSERT OR IGNORE INTO prd_approvals (id,prd_version_id,role,reviewer,status,comment,decided_at,created_at) VALUES (?,?,?,?,?,?,?,?)`,
			approvalID, versionID, role, "", "pending", "", nil, now); err != nil {
			return rollback(fmt.Errorf("创建 %s 审批队列失败：%w", role, err))
		}
	}
	if err := tx.Commit(); err != nil {
		return DeliveryWorkflowState{}, fmt.Errorf("提交 PRD 草案失败：%w", err)
	}
	return s.GetDeliveryWorkflowState()
}

func (s *KnowledgeStore) GetDeliveryWorkflowState() (DeliveryWorkflowState, error) {
	state := DeliveryWorkflowState{Approvals: []DeliveryApproval{}}
	var updatedAt string
	err := s.db.QueryRow(`SELECT i.id,i.initiative_key,v.id,v.version,v.status,v.updated_at
		FROM product_initiatives i JOIN prd_versions v ON v.initiative_id=i.id
		ORDER BY v.updated_at DESC LIMIT 1`).Scan(&state.InitiativeID, &state.InitiativeKey, &state.VersionID, &state.Version, &state.Status, &updatedAt)
	if err == sql.ErrNoRows {
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("读取 PRD 工作流状态失败：%w", err)
	}
	state.Persisted, state.UpdatedAt = true, updatedAt
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM prd_requirements WHERE prd_version_id=?`, state.VersionID).Scan(&state.Requirements); err != nil {
		return state, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM evidence_links e JOIN prd_requirements r ON r.id=e.prd_requirement_id WHERE r.prd_version_id=?`, state.VersionID).Scan(&state.EvidenceLinks); err != nil {
		return state, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM prd_reviews WHERE prd_version_id=?`, state.VersionID).Scan(&state.ReviewCount); err != nil {
		return state, err
	}
	_ = s.db.QueryRow(`SELECT readiness_score,verdict FROM prd_reviews WHERE prd_version_id=? ORDER BY created_at DESC LIMIT 1`, state.VersionID).Scan(&state.ReadinessScore, &state.ReadinessVerdict)
	rows, err := s.db.Query(`SELECT role,reviewer,status,comment,COALESCE(decided_at,'') FROM prd_approvals WHERE prd_version_id=? ORDER BY CASE role WHEN 'product_owner' THEN 1 WHEN 'design_lead' THEN 2 WHEN 'engineering_lead' THEN 3 WHEN 'qa_lead' THEN 4 WHEN 'security_lead' THEN 5 ELSE 99 END`, state.VersionID)
	if err != nil {
		return state, err
	}
	defer rows.Close()
	for rows.Next() {
		var approval DeliveryApproval
		if err := rows.Scan(&approval.Role, &approval.Reviewer, &approval.Status, &approval.Comment, &approval.DecidedAt); err != nil {
			return state, err
		}
		if approval.Status != "approved" {
			state.PendingApprovals++
		}
		state.Approvals = append(state.Approvals, approval)
	}
	return state, rows.Err()
}

// SubmitDeliveryForReview is deliberately gated by the stored Reviewer readiness result.
func (s *KnowledgeStore) SubmitDeliveryForReview() (DeliveryWorkflowState, error) {
	state, err := s.GetDeliveryWorkflowState()
	if err != nil {
		return state, err
	}
	if !state.Persisted {
		return state, fmt.Errorf("请先保存 PRD 草案，再提交研发评审")
	}
	if state.Status != "draft" {
		return state, fmt.Errorf("当前 PRD 版本 %s 状态为 %s，不能重复提交评审", state.Version, state.Status)
	}
	if state.ReadinessScore < 80 {
		return state, fmt.Errorf("Reviewer Readiness 为 %d/100，尚未达到提交研发评审的 80 分门槛；请先补齐缺口", state.ReadinessScore)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`UPDATE prd_versions SET status='in_review',updated_at=? WHERE id=?`, now, state.VersionID); err != nil {
		return state, fmt.Errorf("提交 PRD 评审失败：%w", err)
	}
	if _, err := s.db.Exec(`UPDATE product_initiatives SET status='in_review',updated_at=? WHERE id=?`, now, state.InitiativeID); err != nil {
		return state, fmt.Errorf("更新产品事项评审状态失败：%w", err)
	}
	return s.GetDeliveryWorkflowState()
}

// DecideDeliveryApproval records one role decision after a version has entered formal review.
func (s *KnowledgeStore) DecideDeliveryApproval(role, reviewer, decision, comment string) (DeliveryWorkflowState, error) {
	state, err := s.GetDeliveryWorkflowState()
	if err != nil {
		return state, err
	}
	if !state.Persisted {
		return state, fmt.Errorf("请先保存 PRD 草案")
	}
	if state.Status != "in_review" {
		return state, fmt.Errorf("PRD 版本 %s 尚未进入研发评审，不能记录审批决定", state.Version)
	}
	role = strings.TrimSpace(role)
	if _, ok := map[string]bool{"product_owner": true, "design_lead": true, "engineering_lead": true, "qa_lead": true, "security_lead": true}[role]; !ok {
		return state, fmt.Errorf("未知审批角色：%s", role)
	}
	decision = strings.TrimSpace(decision)
	if decision != "approved" && decision != "rejected" {
		return state, fmt.Errorf("审批决定必须为 approved 或 rejected")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.Exec(`UPDATE prd_approvals SET reviewer=?,status=?,comment=?,decided_at=? WHERE prd_version_id=? AND role=?`,
		shorten(strings.TrimSpace(reviewer), 120), decision, shorten(strings.TrimSpace(comment), 2000), now, state.VersionID, role)
	if err != nil {
		return state, fmt.Errorf("保存审批决定失败：%w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return state, err
	}
	if changed == 0 {
		return state, fmt.Errorf("审批队列中不存在角色：%s", role)
	}
	var pending, rejected int
	if err := s.db.QueryRow(`SELECT SUM(CASE WHEN status='pending' THEN 1 ELSE 0 END),SUM(CASE WHEN status='rejected' THEN 1 ELSE 0 END) FROM prd_approvals WHERE prd_version_id=?`, state.VersionID).Scan(&pending, &rejected); err != nil {
		return state, fmt.Errorf("汇总审批状态失败：%w", err)
	}
	versionStatus := "in_review"
	if rejected > 0 {
		versionStatus = "rejected"
	} else if pending == 0 {
		versionStatus = "approved"
	}
	if _, err := s.db.Exec(`UPDATE prd_versions SET status=?,updated_at=? WHERE id=?`, versionStatus, now, state.VersionID); err != nil {
		return state, fmt.Errorf("更新 PRD 审批状态失败：%w", err)
	}
	if _, err := s.db.Exec(`UPDATE product_initiatives SET status=?,updated_at=? WHERE id=?`, versionStatus, now, state.InitiativeID); err != nil {
		return state, fmt.Errorf("更新产品事项审批状态失败：%w", err)
	}
	return s.GetDeliveryWorkflowState()
}
