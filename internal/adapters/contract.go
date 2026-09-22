package adapters

// ContractVersion is the typed-decision adapter contract every conforming
// adapter implements and the only one the verifier replays. It changes when
// the frozen state, question map, or raw answer contract changes.
const ContractVersion = "0.1.0"
