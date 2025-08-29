// internal/adapters/popflash/models_api.go
package popflash

type getMatchResp struct {
	Match apiMatch `json:"match"`
}

type apiUser struct {
	Name *string `json:"name"`
}

type apiUsersMatch struct {
	Team *int     `json:"team"` // 1 o 2
	User *apiUser `json:"user"`
}

type apiMatch struct {
	ID         int             `json:"id"`
	Map        *string         `json:"map"`
	Datacenter *int            `json:"datacenter"`
	CreatedAt  *string         `json:"created_at"`
	Score1     *int            `json:"score1"`
	Score2     *int            `json:"score2"`
	Users      []apiUsersMatch `json:"users_matches"`
}
