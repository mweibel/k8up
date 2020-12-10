package handler

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vshn/k8up/api/v1alpha1"
	"github.com/vshn/k8up/cfg"
	"github.com/vshn/k8up/job"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"strconv"
	"strings"
	"testing"
)

func TestScheduleHandler_mergeResourcesWithDefaults(t *testing.T) {
	tests := []struct {
		name                        string
		globalCPUResourceLimit      string
		globalCPUResourceRequest    string
		globalMemoryResourceLimit   string
		globalMemoryResourceRequest string
		template                    v1.ResourceRequirements
		resources                   v1.ResourceRequirements
		expected                    v1.ResourceRequirements
	}{
		{
			name:     "Given_NoGlobalDefaults_And_NoScheduleDefaults_When_NoSpec_Then_LeaveEmpty",
			expected: v1.ResourceRequirements{},
		},
		{
			name: "Given_NoGlobalDefaults_And_NoScheduleDefaults_When_Spec_Then_UseSpec",
			resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("50m"),
				},
			},
			expected: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("50m"),
				},
			},
		},
		{
			name: "Given_NoGlobalDefaults_And_ScheduleDefaults_When_NoSpec_Then_ApplyScheduleDefaults",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
		},
		{
			name: "Given_NoGlobalDefaults_And_ScheduleDefaults_When_Spec_Then_UseSpec",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("50m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("50m"),
				},
			},
		},
		{
			name:                        "Given_GlobalDefaults_And_NoScheduleDefaults_When_NoSpec_Then_UseGlobalDefaults",
			globalMemoryResourceRequest: "10Mi",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
				Requests: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse("10Mi"),
				},
			},
		},
		{
			name:                        "Given_GlobalDefaults_And_NoScheduleDefaults_When_Spec_Then_UseSpec",
			globalMemoryResourceRequest: "10Mi",
			resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse("20Mi"),
				},
			},
			expected: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceMemory: resource.MustParse("20Mi"),
				},
			},
		},
		{
			name:                   "Given_GlobalDefaults_And_ScheduleDefaults_When_NoSpec_Then_UseSchedule",
			globalCPUResourceLimit: "10m",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
		},
		{
			name:                   "Given_GlobalDefaults_And_ScheduleDefaults_When_Spec_Then_UseSpec",
			globalCPUResourceLimit: "10m",
			template: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("100m"),
				},
			},
			resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
			expected: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse("200m"),
				},
			},
		},
	}
	cfg.Config = cfg.NewDefaultConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.Config.GlobalCPUResourceLimit = tt.globalCPUResourceLimit
			cfg.Config.GlobalCPUResourceRequest = tt.globalCPUResourceRequest
			cfg.Config.GlobalMemoryResourceLimit = tt.globalMemoryResourceLimit
			cfg.Config.GlobalMemoryResourceRequest = tt.globalMemoryResourceRequest
			require.NoError(t, cfg.Config.ValidateSyntax())
			s := ScheduleHandler{schedule: &v1alpha1.Schedule{Spec: v1alpha1.ScheduleSpec{
				ResourceRequirementsTemplate: tt.template,
			}}}
			result := s.mergeResourcesWithDefaults(tt.resources)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScheduleHandler_randomizeSchedule(t *testing.T) {
	name := "k8up-system/my-scheduled-backup@backup"
	tests := []struct {
		name             string
		schedule         string
		expectedSchedule string
	}{
		{
			name:             "WhenScheduleRandomHourlyGiven_ThenReturnStableRandomizedSchedule",
			schedule:         "@hourly-random",
			expectedSchedule: "52 * * * *",
		},
		{
			name:             "WhenScheduleRandomDailyGiven_ThenReturnStableRandomizedSchedule",
			schedule:         "@daily-random",
			expectedSchedule: "52 4 * * *",
		},
		{
			name:             "WhenScheduleRandomWeeklyGiven_ThenReturnStableRandomizedSchedule",
			schedule:         "@weekly-random",
			expectedSchedule: "52 4 * * 4",
		},
		{
			name:             "WhenScheduleRandomMonthlyGiven_ThenReturnStableRandomizedSchedule",
			schedule:         "@monthly-random",
			expectedSchedule: "52 4 26 * *",
		},
		{
			name:             "WhenScheduleRandomYearlyGiven_ThenReturnStableRandomizedSchedule",
			schedule:         "@yearly-random",
			expectedSchedule: "52 4 26 5 *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ScheduleHandler{
				Config: job.Config{Log: zap.New(zap.UseDevMode(true))},
			}
			result := s.randomizeSchedule(name, tt.schedule)
			assert.Equal(t, tt.expectedSchedule, result)
		})
	}
}

func TestScheduleHandler_getOrGenerateSchedule(t *testing.T) {
	tests := []struct {
		name                 string
		schedule             *v1alpha1.Schedule
		originalSchedule     string
		expectedStatusUpdate bool
		expectedSchedule     string
	}{
		{
			name: "GivenScheduleWithoutStatus_WhenUsingRandomSchedule_ThenPutGeneratedScheduleInStatus",
			schedule: &v1alpha1.Schedule{
				Spec: v1alpha1.ScheduleSpec{
					Backup: &v1alpha1.BackupSchedule{},
				},
			},
			originalSchedule:     "@hourly-random",
			expectedSchedule:     "26 * * * *",
			expectedStatusUpdate: true,
		},
		{
			name: "GivenScheduleWithStatus_WhenUsingRandomSchedule_ThenUseGeneratedScheduleFromStatus",
			schedule: &v1alpha1.Schedule{
				Spec: v1alpha1.ScheduleSpec{
					Backup: &v1alpha1.BackupSchedule{},
				},
				Status: v1alpha1.ScheduleStatus{
					EffectiveSchedules: map[v1alpha1.JobType]string{
						v1alpha1.BackupType: "26 * 3 * *",
					},
				},
			},
			originalSchedule: "@hourly-random",
			expectedSchedule: "26 * 3 * *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ScheduleHandler{
				schedule: tt.schedule,
				Config:   job.Config{Log: zap.New(zap.UseDevMode(true))},
			}
			result := s.getOrGenerateSchedule(v1alpha1.BackupType, tt.originalSchedule)
			assert.Equal(t, tt.expectedSchedule, result)
			assert.Equal(t, tt.expectedStatusUpdate, s.requireStatusUpdate)
			if tt.expectedStatusUpdate {
				assert.Equal(t, tt.expectedSchedule, tt.schedule.Status.EffectiveSchedules[v1alpha1.BackupType])
			}
		})
	}
}

func TestScheduleHandler_randomizeSchedule_VerifyCronSyntax(t *testing.T) {
	s := ScheduleHandler{
		Config: job.Config{Log: zap.New(zap.UseDevMode(true))},
	}
	formatString := "%d is not between %d and %d"
	arr := []int{0, 0, 0, 0, 0}
	for i := 0; i < 200; i++ {
		seed := "namespace/name-" + strconv.Itoa(i) + "@backup"
		schedule := s.randomizeSchedule(seed, "@yearly-random")
		fields := strings.Split(schedule, " ")
		for j, f := range fields {
			number, _ := strconv.Atoi(f)
			arr[j] = number
		}
		assert.InDelta(t, 0, arr[0], 59.0, formatString, arr[0], 0, 59)
		assert.InDelta(t, 0, arr[1], 59.0, formatString, arr[1], 0, 59)
		assert.InDelta(t, 1, arr[2], 26.0, formatString, arr[2], 1, 27)
		assert.InDelta(t, 1, arr[3], 11.0, formatString, arr[3], 1, 12)
	}
	for i := 0; i < 100; i++ {
		seed := "namespace/name-" + strconv.Itoa(i) + "@backup"
		schedule := s.randomizeSchedule(seed, "@weekly-random")
		fields := strings.Split(schedule, " ")
		for j, f := range fields {
			number, _ := strconv.Atoi(f)
			arr[j] = number
		}
		assert.InDelta(t, 0, arr[0], 59.0, formatString, arr[0], 0, 59)
		assert.InDelta(t, 0, arr[1], 59.0, formatString, arr[1], 0, 59)
		assert.InDelta(t, 1, arr[4], 5.0, formatString, arr[4], 0, 6)
	}
}
