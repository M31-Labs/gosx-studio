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
	// Migration is the pending migration plan Promote classifies (spec §8)
	// before recording an intent. The zero value (no steps) classifies Safe:
	// most promotions carry no schema migration at all, and that absence is
	// not itself grounds for a STOP. A caller that DOES know about a pending
	// migration (e.g. an operator-supplied manifest of schema changes) sets
	// Steps explicitly; any Destructive or Unknown step hard-stops Promote
	// with ErrDestructiveMigration.
	Migration MigrationPlan
}

// MigrationClass classifies one migration step's data-safety per spec §8's
// table ("Every schema change introduced here, classified.").
type MigrationClass string

const (
	// MigrationSafe is additive/idempotent/reversible -- promotion proceeds.
	MigrationSafe MigrationClass = "safe"
	// MigrationDestructive drops or rewrites a persisted field/column, or is
	// otherwise irreversible. Promote refuses with ErrDestructiveMigration.
	MigrationDestructive MigrationClass = "destructive"
	// MigrationUnknown is a step Promote cannot classify as Safe. Spec §8:
	// "Any future host change that would drop/rewrite persisted fields is
	// out of scope and a STOP under this descriptor" -- an unclassifiable
	// step must fail closed exactly like a known-destructive one, never be
	// treated as safe by default.
	MigrationUnknown MigrationClass = "unknown"
)

// MigrationStep is one classified change a pending promotion would apply.
type MigrationStep struct {
	Description string
	Class       MigrationClass
}

// MigrationPlan is the full set of migration steps a promotion would apply.
// An empty plan (no steps) classifies Safe: it represents "no migration",
// the common case, not an unclassified one.
type MigrationPlan struct {
	Steps []MigrationStep
}

// Classify reports the plan's overall class: the most severe classification
// among its steps (Destructive outranks Unknown outranks Safe), or Safe when
// there are no steps at all. A step whose Class is anything other than the
// explicit MigrationSafe value -- including the zero value, i.e. a caller
// that forgot to classify a step -- is treated as MigrationUnknown, never as
// safe by default (spec §8: an unclassifiable change is a STOP, not a
// silent pass).
func (p MigrationPlan) Classify() MigrationClass {
	worst := MigrationSafe
	for _, step := range p.Steps {
		switch step.Class {
		case MigrationSafe:
			continue
		case MigrationDestructive:
			return MigrationDestructive
		default:
			worst = MigrationUnknown
		}
	}
	return worst
}

// Blocking reports whether the plan's classification hard-stops Promote.
func (p MigrationPlan) Blocking() bool {
	class := p.Classify()
	return class == MigrationDestructive || class == MigrationUnknown
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
