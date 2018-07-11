package helpers

import "testing"

func TestContains(t *testing.T) {
	type args struct {
		needle   string
		haystack []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "basic match",
			args: args{needle: "test", haystack: []string{"hello", "world", "test"}},
			want: true,
		},
		{
			name: "basic miss",
			args: args{needle: "testx", haystack: []string{"hello", "world", "test"}},
			want: false,
		},
		{
			name: "basic miss",
			args: args{needle: "anything", haystack: make([]string, 0)},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.args.needle, tt.args.haystack); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
