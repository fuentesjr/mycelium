package main

// builtinKind describes one of the five kinds shipped with mycelium.
type builtinKind struct {
	Name             string
	Definition       string
	DefinedAtVersion string
}

// builtinKinds is the registry of kinds that ship with mycelium.
// No kind_definition is required on first use of any of these.
var builtinKinds = []builtinKind{
	{
		Name:             "convention",
		Definition:       "A naming, layout, or structural pattern for organizing data in the store.",
		DefinedAtVersion: "0.1.0",
	},
	{
		Name:             "index",
		Definition:       "A derived or summary file the agent has built or regenerated over a region of the store.",
		DefinedAtVersion: "0.1.0",
	},
	{
		Name:             "archive",
		Definition:       "A region of the store the agent has marked as no-longer-active and moved out of working scope.",
		DefinedAtVersion: "0.1.0",
	},
	{
		Name:             "lesson",
		Definition:       "A distilled insight from past work, intended to inform future behavior.",
		DefinedAtVersion: "0.1.0",
	},
	{
		Name:             "question",
		Definition:       "An open unknown the agent is tracking, expected to resolve into a `lesson` (or be superseded as no-longer-relevant) later.",
		DefinedAtVersion: "0.1.0",
	},
}

// reservedKindDefinition is the meta-kind written by the binary when an agent
// redefines a kind's meaning. It is _-prefixed and therefore reserved; agents
// cannot emit it directly.
const reservedKindDefinition = "_kind_definition"

// isBuiltinKind reports whether name is one of the five built-in kinds.
func isBuiltinKind(name string) bool {
	for _, k := range builtinKinds {
		if k.Name == name {
			return true
		}
	}
	return false
}
