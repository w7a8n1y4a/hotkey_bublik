package game

// Типы данных для REST‑ответов Pepeunit, используемые внутри игры.

type UnitNode struct {
	UUID              string `json:"uuid"`
	Type              string `json:"type"`
	VisibilityLevel   string `json:"visibility_level"`
	IsRewritableInput bool   `json:"is_rewritable_input"`
	TopicName         string `json:"topic_name"`
	CreateDatetime    string `json:"create_datetime"`
	State             string `json:"state"`
	UnitUUID          string `json:"unit_uuid"`
	CreatorUUID       string `json:"creator_uuid"`
}

type Unit struct {
	UUID                     string     `json:"uuid"`
	VisibilityLevel          string     `json:"visibility_level"`
	Name                     string     `json:"name"`
	CreateDatetime           string     `json:"create_datetime"`
	IsAutoUpdateFromRepoUnit bool       `json:"is_auto_update_from_repo_unit"`
	RepoBranch               string     `json:"repo_branch"`
	RepoCommit               string     `json:"repo_commit"`
	UnitStateDict            string     `json:"unit_state_dict"`
	CurrentCommitVersion     string     `json:"current_commit_version"`
	LastUpdateDatetime       string     `json:"last_update_datetime"`
	CreatorUUID              string     `json:"creator_uuid"`
	RepoUUID                 string     `json:"repo_uuid"`
	UnitNodes                []UnitNode `json:"unit_nodes"`
}

type UnitsByNodesResponse struct {
	Count int    `json:"count"`
	Units []Unit `json:"units"`
}

type UnitNodesResponse struct {
	Count     int        `json:"count"`
	UnitNodes []UnitNode `json:"unit_nodes"`
}
