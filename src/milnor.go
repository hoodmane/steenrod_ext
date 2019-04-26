//  File: milnor.go
//  Author: Hood Chatham
//  
//  Defines the basic operations on Milnor algebras. No object oriented code,
//  all functions take basis vectors as inputs and output dictionaries representing
//  Fp-linear combinations of basis vectors. The relevant calls will be wrapped
//  by methods of the MilnorAlgebra and MilnorElement classes in steenrod.py.
//  They are systemtically extended over vector inputs using the
//  @linearextension decorator.
//  
//  The algorithms here are all copied from Sage, in particular from the two
//  files steenrod_algebra_mult.py and (the basis code from) steenrod_algebra_basis.py
//  I have made significant improvements in code legibility and removed all references to Sage code.

package main

import (
    "math"
    "fmt"
)

func main(){
    empty_profile := ProfileList{[]int{},false,false}
    empty_full_profile := FullProfile{empty_profile, empty_profile}
    Alg := func(p int) MilnorAlgebra {
        return MinimalMilnorAlgebra{p,p!=2,empty_full_profile}
    }
    for x := range MilnorBasisGeneric(Alg(3),1){
        fmt.Println(x)
    }
}

type FullProfile struct {
    oddPart Profile
    evenPart Profile
}

type Profile interface {
    getExponent(int, int) int
    getIndex(int) int
    restrictedQ() bool
}

type ProfileList struct {
    profile []int
    truncated bool
    restricted bool
}

type ProfileFunction struct {
    profile_func func(int) int
}

func (P ProfileList) getIndex(index int) int{
    if index < len(P.profile) {
        return P.profile[index]
    } 
    if P.truncated {
        return 0
    } 
    return math.MaxInt32
}

func (P ProfileList) getExponent(p, index int) int{
    if index < len(P.profile) {
        return pow(p, P.profile[index])
    } 
    if P.truncated {
        return 1
    } 
    return math.MaxInt32
}

func (P ProfileList) restrictedQ() bool{
    return P.restricted
}

func (P ProfileFunction) getIndex(index int) int {
    return P.profile_func(index)
}

func (P ProfileFunction) getExponent(p, index int) int {
    n := P.profile_func(index)
    if n >= math.MaxInt32 {
        return n
    } else {
        return pow(p, n)
    }
}

type MilnorElement struct {
    Vector
    generic bool
}

type MilnorAlgebra interface {
    getPrime() int
    genericQ() bool
    getProfile() FullProfile
}

type MinimalMilnorAlgebra struct {
    p int
    generic bool
    profile FullProfile
}

func (A MinimalMilnorAlgebra) getPrime() int {
    return A.p
}

func (A MinimalMilnorAlgebra) genericQ() bool {
    return A.generic
}

func (A MinimalMilnorAlgebra) getProfile() FullProfile {
    return A.profile
}

func NewMilnorZeroVector2(size_hint int) MilnorElement {
    return MilnorElement{NewZeroVector(2, size_hint), false}
}

func NewMilnorZeroVectorGeneric(p, size_hint int) MilnorElement {
    return MilnorElement{NewZeroVector(p, size_hint), true}
}

func NewMilnorBasisVector2(even_part []int) MilnorElement{
    return MilnorElement{NewBasisVector2(even_part), false}
}

func NewMilnorBasisVectorGeneric(p int, odd_part, even_part []int) MilnorElement{
    return MilnorElement{NewBasisVector(p, odd_part, even_part), true}
}


func allocate_milnor_matrix(rows, cols int) [][]int {
    M := make([][]int, rows)
    e := make([]int, rows * cols)
    for i := range M {
        M[i] = e[i * cols:(i + 1) * cols]
    }
    return M    
}

// Initializes an len(r)+1 by len(s)+1 matrix
// Puts r along the first column and s along the first row and zeroes everywhere else.
func initialize_milnor_matrix(r, s []int) [][]int {
    rows := len(r) + 1
    cols := len(s) + 1
    M := allocate_milnor_matrix(rows, cols)
    copy(M[0][1:], s)
    for i, v := range r {
        M[i+1][0] = v
    }
    return M
}

func copy_milnor_matrix_starting_in_row(target, source [][]int, row, cols int) {
    copy(target[0][row*cols:cap(target[0])], source[0][row*cols:cap(source[0])])
}


// This seems to move an i x j block of M back to the first row and column.
// To be honest, I don't really know what the point is, but the milnor_matrices
// function was a little long and this seemed like a decent chunk to extract.
// At least it contains all of the steps that modify M so that seems like a good thing.
func step_milnor_matrix(M [][]int, r, s []int, i, j, x int) [][]int {
    rows := len(r) + 1
    cols := len(s) + 1
    N := allocate_milnor_matrix(rows, cols)
    copy(N[0],M[0])
    for row := 1; row < i; row ++ {
        N[row][0] = r[row-1]
        for col := 1; col < cols; col++ {
            N[0][col] += M[row][col]
        }
    }
    copy_milnor_matrix_starting_in_row(N, M, i, cols)
    for col := 1; col < j; col++ {
        N[0][col] += M[i][col]
        N[i][col] = 0
    }
    N[0][j] --
    N[i][j] ++
    N[i][0] = x
    return N
}

// Generator for Milnor matrices. milnor_product_even iterates over this.
// Uses the same algorithm Monks does in his Maple package to iterate through
// the possible matrices: see
// https://monks.scranton.edu/files/software/Steenrod/steen.html
func milnor_matrices(p int, r, s []int) <-chan [][]int {
    ch := make(chan [][]int)
    go func(){
        defer close(ch)
        rows := len(r) + 1
        cols := len(s) + 1
        M := initialize_milnor_matrix(r, s)
        ch <- M
        for found := true; found ; {
            found = false
            for i := 1; !found && i < rows; i++ {
                total := M[i][0]
                for j := 1; j < cols; j++ {
                    column_above_is_empty := true
                    for k := 0; k < i; k++ {
                        if M[k][j] != 0 {
                            column_above_is_empty = false
                            break
                        }
                    }
                    p_to_the_j := pow(p, j)                    
                    if total < p_to_the_j || column_above_is_empty {
                        total += M[i][j] * p_to_the_j
                    } else {
                        M = step_milnor_matrix(M, r, s, i, j, total - p_to_the_j)
                        found = true
                        ch <- M
                        break 
                    }
                }
            }
        }
    }()
    return ch
}

//Remove trailing zeroes from the list l.
func remove_trailing_zeroes(l []int) []int {
    for i := len(l) - 1; i >= 0; i-- {
        if l[i] != 0 {
            return l[:i+1]
        }
    }
    return l[:0]
}

func max(a, b int) int{
    if a > b {
        return a
    }
    return b
}

func min(a, b int) int{
    if a < b {
        return a
    }
    return b
}

// Handles the multiplication in the even subalgebra of the Steenrod algebra P.
// When p = 2, this is isomorphic to the whole Steenrod algebra so this method does everything.
func MilnorProductEven(p int, r, s []int) MilnorElement {
    result := NewMilnorZeroVectorGeneric(p, -1)
    rows := len(r) + 1
    cols := len(s) + 1
    diags := len(r) + len(s)
    for M := range milnor_matrices(p, r, s) {
        // check diagonals
        coeff := 1
        diagonal_sums := make([]int, diags)
        for n := 1; n <= diags; n++ {
            i_min := max(0, n - cols + 1)
            i_max := min(1 + n, rows)
            nth_diagonal := make([]int, i_max - i_min + 1)
            nth_diagonal_sum := 0
            index := 0
            for i := i_min; i < i_max; i++ {    
                nth_diagonal[index] = M[i][n-i]
                nth_diagonal_sum += nth_diagonal[index]
                index++
            }
            coeff *= Multinomial(nth_diagonal, p)
            coeff = coeff % p
            if coeff == 0 {
                break
            }
            diagonal_sums[n-1] = nth_diagonal_sum
        }
        if coeff != 0 {
            diagonal_sums = remove_trailing_zeroes(diagonal_sums)
            m := Monomial{[]int{}, diagonal_sums}
            result.AddBasisVector(m, coeff)
        }
    }
    return result
}

func inListQ(l []int, n int) bool{
    for _, v := range l {
        if v == n {
            return true
        }
    }
    return false
}


// Reduce m1 * f = (Q_e0 Q_e1 ... P(r1, r2, ...)) * (Q_f0 Q_f1 ...) into the form Sum of Q's * P's
// Result is represented as dictionary of pairs of tuples.
func MilnorProductFullQpart(p int, m1 Monomial, f []int) MilnorElement{
    result := NewMilnorBasisVectorGeneric(p, m1.odd_part, m1.even_part)
    for _, k := range f {
        old_result := result
        result = NewMilnorZeroVectorGeneric(p, -1)
        p_to_the_k := pow(p, k)
        for key, mono := range old_result.GetBasisVectorMap() {
            for i := 0; i < len(mono.even_part) + 1; i++ {
                q_mono := mono.odd_part
                p_mono := mono.even_part
                if inListQ(mono.odd_part, k+i){
                    continue
                }
                // Make sure p_mono[i - 1] is large enough to deduct p^k from it
                if i > 0 && p_mono[i - 1] < p_to_the_k {
                    continue 
                }
                
                if i > 0 {
                    new_p_mono := make([]int, len(p_mono))
                    copy(new_p_mono, p_mono)
                    new_p_mono[i - 1] -= p_to_the_k
                    p_mono = remove_trailing_zeroes(new_p_mono)                
                }
                
                ind := 0
                for _, x := range q_mono {
                    if x >= k+i {
                        ind ++
                    }
                }
                
                // insert(q_mono, len(q_mono) - ind, k+i)
                new_q_mono := make([]int, len(q_mono) + 1)
                copy(new_q_mono, q_mono[:len(q_mono) - ind])
                new_q_mono[len(q_mono) - ind] = k + i
                copy(new_q_mono[len(q_mono) - ind + 1:], q_mono[len(q_mono) - ind:])
                q_mono = new_q_mono
                
                coeff := MinusOneToTheN(ind) * old_result.GetCoeffMap()[key]
                
                result.AddBasisVector(Monomial{q_mono, p_mono}, coeff)
            }
        }
    }
    return result
}    

// Product of Milnor basis elements defined by m1 and m2 at the prime p.
// 
// INPUT:
// 
// - m1 - pair of tuples (e,r), where e is an increasing tuple of
//   non-negative integers and r is a tuple of non-negative integers
// - m2 - pair of tuples (f,s), same format as m1
// - p -- odd prime number
// 
// OUTPUT:
// 
// Dictionary of terms of the form (tuple: coeff), where 'tuple' is
// a pair of tuples, as for r and s, and 'coeff' is an integer mod p.
// 
// This computes the product of the Milnor basis elements
// $Q_{e_1} Q_{e_2} ... P(r_1, r_2, ...)$ and
// $Q_{f_1} Q_{f_2} ... P(s_1, s_2, ...)$.
func MilnorProductFull(p int, m1, m2 Monomial) MilnorElement {
    f := m2.odd_part
    s := m2.even_part
    m1_times_f := MilnorProductFullQpart(p, m1, f)
    // Now for the Milnor matrices.  For each entry '(e,r): coeff' in answer,
    // multiply r with s.  Record coefficient for matrix and multiply by coeff.
    // Store in 'result'.
    if len(s) == 0 {
        return m1_times_f
    }
    
    result := NewMilnorZeroVectorGeneric(p, -1)
    m1_times_f_coeff_map := m1_times_f.GetCoeffMap()
    for key, e_r := range m1_times_f.GetBasisVectorMap() {
        e := e_r.odd_part
        r := e_r.even_part
        coeff := m1_times_f_coeff_map[key]
        prod := MilnorProductEven(p, r, s)
        prod_coeff_map := prod.GetCoeffMap()
        for key, m := range prod.GetBasisVectorMap() {
            m  = Monomial{e, m.even_part}
            c := prod_coeff_map[key]
            result.AddBasisVector(m, coeff*c)
        }
    }
    return result
}

// Multiplication of Milnor basis elements in the non generic case.
func MilnorProduct2(r, s []int) MilnorElement {
    return MilnorProductEven(2, r, s)
}

func MilnorProductGeneric(p int, r, s Monomial) MilnorElement {
    return MilnorProductFull(p, r, s)
}


// Multiply r and s in the Milnor algebra determined by algebra.
// Note that since profile functions determine subalgebras, the product
// doesn't need to care about the profile function at all.
func MilnorProduct(algebra MilnorAlgebra, r, s Monomial) MilnorElement {
    if algebra.genericQ() {
        return MilnorProductFull(algebra.getPrime(), r, s)
    } else {
        return MilnorProductEven(algebra.getPrime(), r.even_part, s.even_part)
    }
}

func CheckOddProfile(profile Profile, q_mono []int) bool {       
    if !profile.restrictedQ() {
        return true
    }
    for _, i := range q_mono {
        if profile.getIndex(i) <= 1 {
            return false
        }
    }
    return true
}

func CheckEvenProfile(p int, profile Profile, exponents []int) bool {
    if !profile.restrictedQ() {
        return true
    }
    for i, exp := range exponents{
        if exp >= profile.getExponent(p, i) {
            return false
        }
    }
    return true
}


// Return the even part of the basis in degree n * 2*(p-1).
// In the nongeneric case, this actually just gets the whole degree n basis.
// Note the factor of two difference between 2*(2-1) and 1.
func MilnorBasisEven(algebra MilnorAlgebra, n int) <-chan []int {
    ch := make(chan []int, 20)
    profile := algebra.getProfile().evenPart
    p := algebra.getPrime()
    go func(){
        defer close(ch)
        if n == 0 {
            ch <- []int{}
            return
        }        
        for exponents := range WeightedIntegerVectors(n, XiDegrees(n, p)) {
            exponents = remove_trailing_zeroes(exponents)
            if CheckEvenProfile(p, profile, exponents) {
                ch <- exponents
            }
        }
    }()
    return ch
}

// Returns the "Q-part" of the basis in degree q_deg.
// This means return the set of monomials Q(i_1) * ... * Q(i_k) where i_1 < ... < i_k
// and the product is in q_deg. Basically it's just an issue of finding partitions of
// q_deg into parts of size |Q(i_j)|, and then there's a profile condition.
func MilnorBasisGenericQpart(algebra MilnorAlgebra, q_deg int) <-chan []int {
    ch := make(chan []int, 20)
    profile := algebra.getProfile().oddPart
    p := algebra.getPrime()
    go func(){
        defer close(ch)
        q_degrees := TauDegrees(q_deg, p)
        for sigma := range RestrictedPartitions(q_deg, q_degrees) {
            // q_mono is the list of indices ocurring in the partition
            q_mono := make([]int, 0, len(q_degrees))
            for idx := 0; idx < len(q_degrees); idx++ {
                if sigma[idx] == 1 {
                    q_mono = append(q_mono, idx)
                }
            }
            if CheckOddProfile(profile, q_mono) {
                ch <- q_mono
            }
        }
    }()
    return ch
}


// Get the basis in degree n for the generic steenrod algebra at the prime p.
// We just put together the "even part" of the basis and the "Q part".
func MilnorBasisGeneric(algebra MilnorAlgebra, n int) <-chan Monomial{
    ch := make(chan Monomial, 20)
    p := algebra.getPrime()
    q := 2*(p - 1)
    go func(){
        defer close(ch)
        if n == 0 {
            ch <- Monomial{[]int {}, []int {}}
            return
        }
        // p_deg records the desired degree of the P part of the basis element.
        // Since p-parts are always divisible by 2p-2, we divide by this first.
        // pow(p, -1) returns 1, so min_q_deg is 0 if q divides n evenly.
        min_q_deg := p_to_the_n_minus_1_over_p_minus_1(p, - 1 + (n % q) )
        for p_deg := 0; p_deg <= n / q; p_deg ++ {
            q_deg := n - p_deg * q
    
            // if this inequality holds, no way to have a partition
            // with distinct parts.
            if q_deg < min_q_deg {
                break
            }
    
            Q_parts := MilnorBasisGenericQpart(algebra, q_deg)
            P_parts := MilnorBasisEven(algebra, p_deg)
            for q_part := range Q_parts {
                for p_part := range P_parts {
                    fmt.Println("p:", p_part)
                    ch <- Monomial{q_part, p_part}
                }
            }
        }
    }()
    return ch
}
// 
// 
// def basis(n, *, algebra):
//     r"""
//     Milnor basis in dimension `n` with profile function ``profile``.
// 
//     INPUT:
// 
//     - ``n`` - non-negative integer
// 
//     - ``p`` - positive prime number
// 
//     - ``profile`` - profile function (optional, default None).
//       Together with ``truncation_type``, specify the profile function
//       to be used; None means the profile function for the entire
//       Steenrod algebra.  See
//       :mod:`sage.algebras.steenrod.steenrod_algebra` and
//       :func:`SteenrodAlgebra <sage.algebras.steenrod.steenrod_algebra.SteenrodAlgebra>`
//       for information on profile functions.
// 
//     - ``truncation_type`` - truncation type, either 0 or Infinity
//       (optional, default Infinity if no profile function is specified,
//       0 otherwise)
// 
//     OUTPUT: tuple of mod p Milnor basis elements in dimension n
// 
//     At the prime 2, the Milnor basis consists of symbols of the form
//     `\text{Sq}(m_1, m_2, ..., m_t)`, where each
//     `m_i` is a non-negative integer and if `t>1`, then
//     `m_t \neq 0`. At odd primes, it consists of symbols of the
//     form `Q_{e_1} Q_{e_2} ... P(m_1, m_2, ..., m_t)`,
//     where `0 \leq e_1 < e_2 < ...`, each `m_i` is a
//     non-negative integer, and if `t>1`, then
//     `m_t \neq 0`.
// 
//     EXAMPLES::
// 
//         sage: import milnor
//         sage: milnor.basis(7)
//         ((0, 0, 1), (1, 2), (4, 1), (7,))
//         sage: milnor.basis(7, 2)
//         ((0, 0, 1), (1, 2), (4, 1), (7,))
//         sage: milnor.basis(4, 2)
//         ((1, 1), (4,))
//         sage: milnor.basis(4, 2, profile=[2,1])
//         ((1, 1),)
//         sage: milnor.basis(4, 2, profile=(), truncation_type=0)
//         ()
//         sage: milnor.basis(4, 2, profile=(), truncation_type=Infinity)
//         ((1, 1), (4,))
//         sage: milnor.basis(9, 3)
//         (((1,), (1,)), ((0,), (2,)))
//         sage: milnor.basis(17, 3)
//         (((2,), ()), ((1,), (3,)), ((0,), (0, 1)), ((0,), (4,)))
//         sage: milnor.basis(48, p=5)
//         (((), (0, 1)), ((), (6,)))
//         sage: len(milnor.basis(100,3))
//         13
//         sage: len(milnor.basis(200,7))
//         0
//         sage: len(milnor.basis(240,7))
//         3
//         sage: len(milnor.basis(240,7, profile=((),()), truncation_type=Infinity))
//         3
//         sage: len(milnor.basis(240,7, profile=((),()), truncation_type=0))
//         0
//     """
//     if algebra.generic:
//         return basis_generic(n, algebra.p, algebra.profile)
//     else:
//         return basis_even(n, 2, algebra.profile)

