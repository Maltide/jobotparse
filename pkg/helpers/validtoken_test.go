package helpers

import (
	"testing"
	"time"

	"github.com/Maltide/jobotparse/pkg/types"
	"go.uber.org/zap"
)

func TestIsValidToken(t *testing.T) {
	log := zap.NewNop().Sugar()

	now := time.Now().Unix()

	expiresIn := int64(1000)

	negativeExpiresIn := int64(-1000)

	tests := []struct {
		name    string
		tokens  *types.Client
		want    bool
		wantErr bool
	}{
		{
			name: "valid token",
			tokens: &types.Client{
				AccessToken:  "valid_access_token",
				RefreshToken: "valid_refresh_token",
				ExpiresIn:    int(expiresIn), // future time
				Ttl:          int(now + int64(expiresIn)),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "expired token",
			tokens: &types.Client{
				AccessToken:  "some token",
				RefreshToken: "some refresh token",
				ExpiresIn:    int(negativeExpiresIn),       // past time,
				Ttl:          int(now + negativeExpiresIn), // past time,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "empty tokens",
			tokens: &types.Client{
				AccessToken:  "",
				RefreshToken: "",
				ExpiresIn:    int(expiresIn),
				Ttl:          int(now + expiresIn),
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsValidToken(tt.tokens, log)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error: %v, wantErr: %v, got:%v, want:%v", err, tt.wantErr, got, tt.want)
			}

			if !tt.wantErr && got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
