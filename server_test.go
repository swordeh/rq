package main

import (
	"testing"
)

//	func TestQueueMediaHandler(t *testing.T) {
//		type args struct {
//			w   http.ResponseWriter
//			req *http.Request
//		}
//		tests := []struct {
//			name string
//			args args
//		}{
//			// TODO: Add test cases.
//		}
//		for _, tt := range tests {
//			t.Run(tt.name, func(t *testing.T) {
//				QueueMediaHandler(tt.args.w, tt.args.req)
//			})
//		}
//	}
//func TestReturnHTTPErrorResponse(t *testing.T) {
//	type args struct {
//		w            http.ResponseWriter
//		errorMessage string
//		status       int
//	}
//	tests := []struct {
//		name string
//		args args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			ReturnHTTPErrorResponse(tt.args.w, tt.args.errorMessage, tt.args.status)
//		})
//	}
//}

func Test_contains(t *testing.T) {
	type args struct {
		slice []string
		str   string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "has a record",
			args: args{
				slice: []string{"ice", "vodka", "espresso", "kahlua"},
				str:   "vodka",
			},
			want: true,
		},
		{
			name: "has no record",
			args: args{
				slice: []string{"whiskey", "lemon", "syrup", "egg white"},
				str:   "gin",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.args.slice, tt.args.str); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
