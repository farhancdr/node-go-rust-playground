package custom_validator

import (
	"testing"
)

func TestCustom_Validator(t *testing.T) {
	type args struct {
		s any
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test1",
			args: args{
				s: GameLibraryTagDto{
					GameLibraryTagId: "123",
					TagName:          "Shooter",
					CategoryName:     "GAME_CATEGORIES",
				},
			},
			wantErr: false,
		},
		{
			name: "Test2",
			args: args{
				s: GameLibraryTagDto{
					GameLibraryTagId: "123",
					TagName:          "Shooter",
					CategoryName:     "INVALID_CATEGORY",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateStruct(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("Custom_Validator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
