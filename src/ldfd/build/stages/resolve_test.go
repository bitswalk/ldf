package stages

import (
	"testing"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

func TestIsComponentCompatible(t *testing.T) {
	tests := []struct {
		name       string
		supported  []db.TargetArch
		targetArch db.TargetArch
		want       bool
	}{
		{
			name:       "empty list supports all",
			supported:  nil,
			targetArch: db.ArchX86_64,
			want:       true,
		},
		{
			name:       "matching architecture",
			supported:  []db.TargetArch{db.ArchX86_64, db.ArchAARCH64},
			targetArch: db.ArchAARCH64,
			want:       true,
		},
		{
			name:       "single matching architecture",
			supported:  []db.TargetArch{db.ArchX86_64},
			targetArch: db.ArchX86_64,
			want:       true,
		},
		{
			name:       "no match",
			supported:  []db.TargetArch{db.ArchAARCH64},
			targetArch: db.ArchX86_64,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &db.Component{SupportedArchitectures: tt.supported}
			got := isComponentCompatible(c, tt.targetArch)
			if got != tt.want {
				t.Errorf("isComponentCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}
