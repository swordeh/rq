package helpers

import (
    "testing"
)

func TestContains(t *testing.T) {
    tests := []struct {
        name    string
        slice   []string
        str     string
        want    bool
    }{
        {
            name: "Case Contains",
            slice: []string{"apple", "banana", "cherry"},
            str: "banana",
            want: true,
        },
        {
            name: "Case Not Contains",
            slice: []string{"apple", "banana", "cherry"},
            str: "pear",
            want: false,
        },
        {
            name: "Case Empty Slice",
            slice: []string{},
            str: "apple",
            want: false,
        },
        {
            name: "Case Nil Slice",
            slice: nil,
            str: "banana",
            want: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Contains(&tt.slice, tt.str) 
            if got != tt.want {
                t.Errorf("TestContains() = %v, want %v", got, tt.want)
            }
        })
    }
}