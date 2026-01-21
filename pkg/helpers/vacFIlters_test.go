package helpers

import (
	"net/http"
	"testing"

	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

func TestVacancyFilters(t *testing.T) {
	lg := zap.NewNop().Sugar()

	type args struct {
		url    string
		nilReq bool
	}

	tests := []struct {
		name    string
		args    args
		want    types.Filters
		wantErr bool
	}{
		{
			name: "valid params",
			args: args{url: "/vacancies?profession=developer&town=Москва&salary_from=50000"},
			want: types.Filters{
				Profession: "developer",
				Town:       "Москва",
				SalaryFrom: "50000",
			},
			wantErr: false,
		},
		{
			name:    "empty params",
			args:    args{url: "/vacancies"},
			want:    types.Filters{},
			wantErr: true,
		},
		{
			name:    "nil request",
			args:    args{nilReq: true},
			want:    types.Filters{},
			wantErr: true,
		},
		{
			name:    "missing profession",
			args:    args{url: "/vacancies?town=Москва&salary_from=50000"},
			want:    types.Filters{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error
			if tt.args.nilReq {
				req = nil
			} else {
				req, err = http.NewRequest("GET", tt.args.url, nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}
			}

			got, err := VacancyFilters(req, lg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v, got %v, want %v", err, tt.wantErr, got, tt.want)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
