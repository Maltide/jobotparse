package helpers

import (
	"testing"

	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDBWrite(t *testing.T) {
	log := zap.NewNop().Sugar()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}

	db.AutoMigrate(&types.Vacancy{})

	tests := []struct {
		name      string
		vacancies types.VacanciesResponse
		want      types.VacanciesResponse
		wantErr   bool
	}{
		{
			name: "simple vacancy",
			vacancies: types.VacanciesResponse{
				Objects: []types.Vacancy{
					{
						ExternalID: 1,
						Profession: "developer",
						Moveable:   true,
						Town:       &types.Town{Title: "New York"},
					},
				},
			},
			want: types.VacanciesResponse{
				Objects: []types.Vacancy{
					{
						ExternalID: 1,
						Profession: "developer",
						Moveable:   true,
						TownName:   "New York",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "vacancy with nil town",
			vacancies: types.VacanciesResponse{
				Objects: []types.Vacancy{
					{
						ExternalID: 2,
						Profession: "designer",
						Moveable:   false,
						Town:       nil,
					},
				},
			},
			want: types.VacanciesResponse{
				Objects: []types.Vacancy{
					{
						ExternalID: 2,
						Profession: "designer",
						Moveable:   false,
						TownName:   "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple vacancies",
			vacancies: types.VacanciesResponse{
				Objects: []types.Vacancy{
					{
						ExternalID: 3,
						Profession: "manager",
						Moveable:   true,
						Town:       &types.Town{Title: "Los Angeles"},
					},
					{
						ExternalID: 4,
						Profession: "analyst",
						Moveable:   false,
						Town:       nil,
					},
				},
			},
			want: types.VacanciesResponse{
				Objects: []types.Vacancy{
					{
						ExternalID: 3,
						Profession: "manager",
						Moveable:   true,
						TownName:   "Los Angeles",
					},
					{
						ExternalID: 4,
						Profession: "analyst",
						Moveable:   false,
						TownName:   "",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty vacancies",
			vacancies: types.VacanciesResponse{
				Objects: []types.Vacancy{},
			},
			want: types.VacanciesResponse{
				Objects: []types.Vacancy{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DBWrite(db, tt.vacancies, log)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DBWrite() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("DBWrite() returned unexpected error: %v", err)
			}
			for i, got := range tt.vacancies.Objects {
				want := tt.want.Objects[i]
				if got.ExternalID != want.ExternalID ||
					got.Profession != want.Profession ||
					got.Moveable != want.Moveable ||
					got.TownName != want.TownName {
					t.Fatalf("DBWrite() got = %+v, want %+v", got, want)
				}
			}
		})
	}
}
