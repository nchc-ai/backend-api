package image

type General struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
}

type ImageInfo struct {
	User           string `json:"user"`
	Name           string `json:"name"`
	Namespace      string `json:"namespace"`
	RepositoryType string `json:"repository_type"`
	Status         int    `json:"status"`
	Description    string `json:"description"`
	IsPrivate      bool   `json:"is_private"`
	IsAutomated    bool   `json:"is_automated"`
	CanEdit        bool   `json:"can_edit"`
	StarCount      int    `json:"star_count"`
	PullCount      int    `json:"pull_count"`
	LastUpdated    string `json:"last_updated"`
	IsMigrated     bool   `json:"is_migrated"`
}

type TagInfo struct {
	Creator             int     `json:"creator"`
	Id                  int     `json:"id"`
	ImageId             *string `json:"image_id"`
	Images              []Image `json:"images"`
	LastUpdated         string  `json:"last_updated"`
	LastUpdater         int     `json:"last_updater"`
	LastUpdatedUsername string  `json:"last_updater_username"`
	Repository          int     `json:"repository"`
	Name                string  `json:"name"`
	FullSize            int     `json:"full_size"`
	V2                  bool    `json:"v2"`
}

type Image struct {
	Architecture string  `json:"architecture"`
	Features     string  `json:"features"`
	Variant      *string `json:"variant"`
	Digest       string  `json:"digest"`
	Os           string  `json:"os"`
	OsFeatures   string  `json:"os_features"`
	OsVersion    *string `json:"os_version"`
	Size         int     `json:"size"`
}

type ImageResult struct {
	General
	Results []ImageInfo `json:"results"`
}

type TagResult struct {
	General
	Results []TagInfo `json:"results"`
}
