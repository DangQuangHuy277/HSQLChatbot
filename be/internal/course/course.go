package course

type Course struct {
	ID             int    `json:"id" db:"id"`
	Code           string `json:"code" db:"code"`
	Name           string `json:"name" db:"name"`
	EnglishName    string `json:"english_name" db:"english_name"`
	Credits        int    `json:"credits" db:"credits"`
	PracticeHours  int    `json:"practice_hours" db:"practice_hours"`
	TheoryHours    int    `json:"theory_hours" db:"theory_hours"`
	SelfLearnHours int    `json:"self_learn_hours" db:"self_learn_hours"`
	Prerequisite   *int   `json:"prerequisite,omitempty" db:"prerequisite"`
}
