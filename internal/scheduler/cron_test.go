package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCron(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
		check   func(*testing.T, *CronSchedule)
	}{
		{
			name:    "invalid number of fields",
			expr:    "* * *",
			wantErr: true,
		},
		{
			name: "all stars",
			expr: "* * * * *",
			check: func(t *testing.T, c *CronSchedule) {
				for i := 0; i <= 59; i++ {
					assert.True(t, c.Minute[i], "minute %d should be set", i)
				}
				for i := 0; i <= 23; i++ {
					assert.True(t, c.Hour[i], "hour %d should be set", i)
				}
				for i := 1; i <= 31; i++ {
					assert.True(t, c.Day[i], "day %d should be set", i)
				}
				for i := 1; i <= 12; i++ {
					assert.True(t, c.Month[i], "month %d should be set", i)
				}
				for i := 0; i <= 6; i++ {
					assert.True(t, c.Weekday[i], "weekday %d should be set", i)
				}
			},
		},
		{
			name: "single values",
			expr: "1 2 3 4 5",
			check: func(t *testing.T, c *CronSchedule) {
				assert.True(t, c.Minute[1])
				assert.True(t, c.Hour[2])
				assert.True(t, c.Day[3])
				assert.True(t, c.Month[4])
				assert.True(t, c.Weekday[5])
			},
		},
		{
			name: "lists",
			expr: "1,15,30 0,12 1,15,30 1,6,12 0,3,6",
			check: func(t *testing.T, c *CronSchedule) {
				assert.True(t, c.Minute[1] && c.Minute[15] && c.Minute[30])
				assert.True(t, c.Hour[0] && c.Hour[12])
				assert.True(t, c.Day[1] && c.Day[15] && c.Day[30])
				assert.True(t, c.Month[1] && c.Month[6] && c.Month[12])
				assert.True(t, c.Weekday[0] && c.Weekday[3] && c.Weekday[6])
			},
		},
		{
			name: "ranges",
			expr: "1-5 2-4 10-15 3-6 0-2",
			check: func(t *testing.T, c *CronSchedule) {
				for i := 1; i <= 5; i++ {
					assert.True(t, c.Minute[i])
				}
				for i := 2; i <= 4; i++ {
					assert.True(t, c.Hour[i])
				}
				for i := 10; i <= 15; i++ {
					assert.True(t, c.Day[i])
				}
				for i := 3; i <= 6; i++ {
					assert.True(t, c.Month[i])
				}
				for i := 0; i <= 2; i++ {
					assert.True(t, c.Weekday[i])
				}
			},
		},
		{
			name:    "invalid minute",
			expr:    "60 * * * *",
			wantErr: true,
		},
		{
			name:    "invalid hour",
			expr:    "* 24 * * *",
			wantErr: true,
		},
		{
			name:    "invalid day",
			expr:    "* * 32 * *",
			wantErr: true,
		},
		{
			name:    "invalid month",
			expr:    "* * * 13 *",
			wantErr: true,
		},
		{
			name:    "invalid weekday",
			expr:    "* * * * 7",
			wantErr: true,
		},
		{
			name:    "invalid range format",
			expr:    "1-2-3 * * * *",
			wantErr: true,
		},
		{
			name:    "invalid range values",
			expr:    "5-1 * * * *",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCron(tt.expr)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.check(t, got)
		})
	}
}

func TestCronSchedule_Next(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		after    time.Time
		want     time.Time
	}{
		{
			name:     "every minute",
			schedule: "* * * * *",
			after:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			want:     time.Date(2024, 1, 1, 0, 1, 0, 0, time.UTC),
		},
		{
			name:     "specific minute",
			schedule: "30 * * * *",
			after:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			want:     time.Date(2024, 1, 1, 0, 30, 0, 0, time.UTC),
		},
		{
			name:     "specific hour and minute",
			schedule: "45 12 * * *",
			after:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			want:     time.Date(2024, 1, 1, 12, 45, 0, 0, time.UTC),
		},
		{
			name:     "specific day of month",
			schedule: "0 0 15 * *",
			after:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			want:     time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "specific weekday",
			schedule: "0 0 * * 0", // Every Sunday at midnight
			after:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // Monday
			want:     time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC), // Next Sunday
		},
		{
			name:     "month rollover",
			schedule: "0 0 1 * *", // First of every month
			after:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			want:     time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "year rollover",
			schedule: "0 0 1 1 *", // January 1st
			after:    time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			want:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := ParseCron(tt.schedule)
			require.NoError(t, err)
			got := c.Next(tt.after)
			assert.Equal(t, tt.want, got)
		})
	}
} 