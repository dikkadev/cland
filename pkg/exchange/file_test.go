package exchange

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name string
		args args
		want *Notification
	}{
		{
			name: "regular",
			args: args{
				lines: []string{
					"topic",
					"key1: value1",
					"---",
					"message",
				},
			},
			want: &Notification{
				Topic: "topic",
				Metadata: map[string]string{
					"key1": "value1",
				},
				Message: "message",
			},
		},
		{
			name: "empty metadata",
			args: args{
				lines: []string{
					"topic",
					"---",
					"message",
				},
			},
			want: &Notification{
				Topic:    "topic",
				Metadata: map[string]string{},
				Message:  "message",
			},
		},
		{
			name: "complex metadata",
			args: args{
				lines: []string{
					"topic",
					"data: {\"key\": \"value\"}",
					"---",
					"message",
				},
			},
			want: &Notification{
				Topic: "topic",
				Metadata: map[string]string{
					"data": "{\"key\": \"value\"}",
				},
				Message: "message",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := parse(tt.args.lines)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "no topic",
			args: args{
				lines: []string{
					"---",
					"message",
				},
			},
			want: &NoTopicError{},
		},
		{
			name: "empty message",
			args: args{
				lines: []string{
					"topic",
					"key1: value1",
					"---",
				},
			},
			want: &EmptyMessageError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parse(tt.args.lines)
			if err == nil {
				t.Errorf("parse() expected error, got nil")
			} else if reflect.TypeOf(err) != reflect.TypeOf(tt.want) {
				t.Errorf("parse() error = %v, want %v", reflect.TypeOf(err), reflect.TypeOf(tt.want))
			}
		})
	}
}

func TestParseMetadata(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "regular",
			args: args{
				lines: []string{
					"key1: value1",
					"key2: value2",
				},
			},
			want: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name: "empty",
			args: args{
				lines: []string{},
			},
			want: map[string]string{},
		},
		{
			name: "comment",
			args: args{
				lines: []string{
					"key1: value1",
					"-- comment",
					"key2: value2",
				},
			},
			want: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
	}
	for _, tt := range tests {
		t.Run("good_"+tt.name, func(t *testing.T) {
			if got := parseMetadata(tt.args.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}
