package engine

import (
	"errors"
	"time"
)

// EnvironmentKind distinguishes the always-on editor target from the
// production-shaped promotion target.
type EnvironmentKind string

const (
	// EnvStaging is the live editor target.
	EnvStaging EnvironmentKind = "staging"
	// EnvProduction is the production-shaped promotion target. It stays
	// Configured=false until an operator explicitly configures it.
	EnvProduction EnvironmentKind = "production"
)

// Environment is one immutable-digest-addressed deploy target.
type Environment struct {
	Key                string
	Kind               EnvironmentKind
	Configured         bool
	ImageDigest        string // sha256:... immutable
	DataRestorePointID string // restore point that pins the data at promotion time
}

// PromotionIntent is the recorded intent Promote emits. It is v1 an
// operator-run runbook trigger, never a k8s call: SeedData is
// compile-time-fixed false because promotion never seeds/resets data.
type PromotionIntent struct {
	ID             string
	Target         TargetRef
	FromDigest     string
	ToDigest       string
	RestorePointID string
	Actor          string
	SeedData       bool
	Created        time.Time
}

// PromoteCommand requests a promotion. It never touches k8s/shell; Promote
// only classifies + records the intent (see spec §7).
type PromoteCommand struct {
	Target      TargetRef
	Actor       Actor
	Environment Environment
	ToDigest    string
	OperationID string
}

// ErrProductionNotConfigured is returned by Promote when the target
// environment is production-shaped but has not been explicitly configured
// and authorized (matches infra README: "Do not attach production hosts to
// this manifest.").
var ErrProductionNotConfigured = errors.New("production environment is not configured")

// ErrDestructiveMigration is the hard STOP Promote returns when the pending
// migration cannot be classified as safe. Migration classification lands in
// a later slice (spec §8 / S8); Promote does not yet return this error.
var ErrDestructiveMigration = errors.New("destructive or unknown migration: promotion refused")
