package entities

type NewsSentiment struct {
	NewsHeadline    string
	Url             string
	ArticleBody     string
	ArticleBodyHtml string
	DatePublished   string
	PositiveScore   float64
	NegativeScore   float64
	NeutralScore    float64
}
