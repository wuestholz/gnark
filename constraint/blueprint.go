package constraint

import (
	"fmt"
	"math/rand"
)

type BlueprintID uint32

// Blueprint enable representing heterogeneous constraints or instructions in a constraint system
// in a memory efficient way. Blueprints essentially help the frontend/ to "compress"
// constraints or instructions, and specify for the solving (or zksnark setup) part how to
// "decompress" and optionally "solve" the associated wires.
type Blueprint interface {
	// CalldataSize return the number of calldata input this blueprint expects.
	// If this is unknown at compile time, implementation must return -1 and store
	// the actual number of inputs in the first index of the calldata.
	CalldataSize() int

	// NbConstraints return the number of constraints this blueprint creates.
	NbConstraints() int

	// NbOutputs return the number of output wires this blueprint creates.
	NbOutputs(inst Instruction) int

	// UpdateInstructionTree updates the instruction tree;
	// since the blue print knows which wires it references, it updates
	// the instruction tree with the level of the (new) wires.
	UpdateInstructionTree(inst Instruction, tree InstructionTree) Level
}

var MainFaultInjector = newFaultInjector(0, 1, 0.0001)

type faultInjector struct {
	rnd       *rand.Rand
	faultProb float64
	maxFaults int
	numFaults int
	numCalls  int
}

func newFaultInjector(seed int64, maxFaults int, faultProb float64) *faultInjector {
	return &faultInjector{
		rnd:       rand.New(rand.NewSource(seed)),
		faultProb: faultProb,
		maxFaults: maxFaults,
	}
}

func (r *faultInjector) Reset(seed int64, maxFaults int, faultProb float64) {
	r.rnd = rand.New(rand.NewSource(seed))
	r.faultProb = faultProb
	r.maxFaults = maxFaults
	r.numFaults = 0
	r.numCalls = 0
}

func (r *faultInjector) MaxFaults() int {
	return r.maxFaults
}

func (r *faultInjector) NumFaults() int {
	return r.numFaults
}

func (r *faultInjector) NumCalls() int {
	return r.numCalls
}

func (r *faultInjector) Delta() int64 {
	r.numCalls++
	if r.maxFaults <= r.numFaults {
		return 0
	}
	if r.rnd.Float64() < r.faultProb {
		d := int64(1)
		if r.rnd.Float64() < 0.5 {
			d = -1
		}
		fmt.Printf("faultInjector: %d\n", d)
		r.numFaults++
		return d
	}
	return 0
}

// Solver represents the state of a constraint system solver at runtime. Blueprint can interact
// with this object to perform run time logic, solve constraints and assign values in the solution.
type Solver interface {
	Field

	GetValue(cID, vID uint32) Element
	GetCoeff(cID uint32) Element
	SetValue(vID uint32, f Element)
	IsSolved(vID uint32) bool

	// Read interprets input calldata as a LinearExpression,
	// evaluates it and return the result and the number of uint32 word read.
	Read(calldata []uint32) (Element, int)
}

// BlueprintSolvable represents a blueprint that knows how to solve itself.
type BlueprintSolvable interface {
	Blueprint
	// Solve may return an error if the decoded constraint / calldata is unsolvable.
	Solve(s Solver, instruction Instruction) error
}

// BlueprintR1C indicates that the blueprint and associated calldata encodes a R1C
type BlueprintR1C interface {
	Blueprint
	CompressR1C(c *R1C, to *[]uint32)
	DecompressR1C(into *R1C, instruction Instruction)
}

// BlueprintSparseR1C indicates that the blueprint and associated calldata encodes a SparseR1C.
type BlueprintSparseR1C interface {
	Blueprint
	CompressSparseR1C(c *SparseR1C, to *[]uint32)
	DecompressSparseR1C(into *SparseR1C, instruction Instruction)
}

// BlueprintHint indicates that the blueprint and associated calldata encodes a hint.
type BlueprintHint interface {
	Blueprint
	CompressHint(h HintMapping, to *[]uint32)
	DecompressHint(h *HintMapping, instruction Instruction)
}

// BlueprintStateful indicates that the blueprint can be reset to its initial state.
type BlueprintStateful interface {
	BlueprintSolvable

	// Reset is called by the solver between invocation of Solve.
	Reset()
}

// Compressible represent an object that knows how to encode itself as a []uint32.
type Compressible interface {
	// Compress interprets the objects as a LinearExpression and encodes it as a []uint32.
	Compress(to *[]uint32)
}
