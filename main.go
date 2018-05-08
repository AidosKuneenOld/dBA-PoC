// Copyright (c) 2018 Aidos Developer

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/tmc/dot"
)

//byzanthine fault tolerance
const bft int = 1

//number of validators
const validators int = 3*bft + 1

//threshold of validators that have to confirm a tx for it to be valid
const threshold int = validators - bft

//random dag
var ranDag bool

type tx struct {
	prevtx1  *tx
	prevtx2  *tx
	validity [validators]bool //referenced by validator n : true/false // 1st reference / 2nd reference for double spent
	confirm  bool
	count    int // tx with same count are double spents
}

type vtx struct {
	prev  *vtx
	tx    *tx
	count int
}

//save ds for graph
var dsG []*tx

//append n new tx randomly to simulate dag behaviour
func newTx(n int, txList []*tx) []*tx {
	length := len(txList)
	for i := length; i < n+length; i++ {
		nLength := len(txList)
		index1 := random(0, nLength)
		index2 := random(0, nLength)
		for index1 == index2 {
			index2 = random(0, nLength)
		}
		tx1 := txList[index1]
		tx2 := txList[index2]
		tx := tx{tx1, tx2, [validators]bool{}, false, i}
		//fmt.Printf("newTx: %v, confirmed: %v, references: %v,%v \n", tx.count, tx.confirm, tx.prevtx1.count, tx.prevtx2.count)
		txList = append(txList, &tx)
	}
	return txList
}

//insert/create a double spent (tx with same count)
func insertDS(txList []*tx) []*tx {
	//select a tx randomly (>1 to not select genesis tx)
	nLength := len(txList)
	ran := random(2, nLength)
	//choose "tips" randomly
	index1 := random(0, nLength)
	index2 := random(0, nLength)
	for index1 == index2 && (ran == index1 || ran == index2) {
		index2 = random(0, nLength)
		if ran == index1 {
			index1 = random(0, nLength)
		}
	}
	tx1 := txList[index1]
	tx2 := txList[index2]
	tx := tx{tx1, tx2, [validators]bool{}, false, ran}
	dsG = append(dsG, &tx)
	//fmt.Printf("DS: %v, confirmed: %v, references: %v,%v \n", tx.count, tx.confirm, tx.prevtx1.count, tx.prevtx2.count)
	txList = append(txList, &tx)
	return txList
}

func random(min, max int) int {
	if ranDag {
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		return r1.Intn(max-min) + min
	} else {
		return rand.Intn(max-min) + min
	}

}

//return all tips
func getTips(txList []*tx) []*tx {
	noTip := make(map[*tx]struct{})
	var tip []*tx
	// gather all tx numbers
	for _, tx := range txList {
		if tx.prevtx1 != nil && tx.prevtx2 != nil {
			noTip[tx.prevtx1] = struct{}{}
			noTip[tx.prevtx2] = struct{}{}
		}

	}
	// check for tips
	for _, tx := range txList {
		if _, exists := noTip[tx]; !exists {
			tip = append(tip, tx)
			//fmt.Printf("tips: %v, confirmed: %v, references: %v,%v \n", tx.count, tx.confirm, tx.prevtx1.count, tx.prevtx2.count)
		}
	}

	return tip
}

//confirm recursively
func confirm(tx1 *tx, validator int, ds map[tx]struct{}, ref map[int]*tx) {
	if _, exists := ds[*tx1]; !exists {
		ds[*tx1] = struct{}{}
		ref[tx1.count] = tx1
		tx1.validity[validator] = true
	}
	reached := false
	for !reached {
		if tx1.prevtx1 != nil {
			if _, exists := ds[*tx1.prevtx1]; !exists {
				ds[*tx1.prevtx1] = struct{}{}
				ref[tx1.prevtx1.count] = tx1.prevtx1
				tx1.prevtx1.validity[validator] = true
				confirm(tx1.prevtx1, validator, ds, ref)
			}
		}
		if tx1.prevtx2 != nil {
			if _, exists := ds[*tx1.prevtx2]; !exists {
				ds[*tx1.prevtx2] = struct{}{}
				ref[tx1.prevtx2.count] = tx1.prevtx2
				tx1.prevtx2.validity[validator] = true
				confirm(tx1.prevtx2, validator, ds, ref)
			}
		}
		reached = true
	}
}

//traverse to check for DS
func traverse(tx1 tx, validator int, ds map[tx]struct{}, count map[int]struct{}, result *bool) {
	//fmt.Println(tx1.count)
	if !tx1.validity[validator] {
		if _, exists := ds[tx1]; !exists {
			if _, exists := count[tx1.count]; exists {
				//unconfirmed ds
				fmt.Printf("Unconfirmed DS detected can't confirm tip: %v \n", tx1.prevtx1.count)
				*result = false
			}
			//fmt.Printf("Tip: %v %v %v \n", tx1.count, tx1.prevtx1.count, tx1.prevtx2.count)
			ds[tx1] = struct{}{}
			count[tx1.count] = struct{}{}
		}
	}
	reached := false
	for !reached {
		// only need to traverse to a already confirmed tx
		if tx1.prevtx1 != nil && !tx1.prevtx1.validity[validator] {
			if _, exists := ds[*tx1.prevtx1]; !exists {
				if _, exists := count[tx1.prevtx1.count]; exists {
					//unconfirmed ds
					fmt.Printf("Unconfirmed DS1 detected can't confirm tip: %v %v %v \n", tx1.count, tx1.prevtx1.count, tx1.prevtx2.count)
					*result = false
				}
				//fmt.Printf("Tip1: %v \n", tx1.prevtx1.count)
				count[tx1.prevtx1.count] = struct{}{}
				ds[*tx1.prevtx1] = struct{}{}
				traverse(*tx1.prevtx1, validator, ds, count, result)
			}
		}
		if tx1.prevtx2 != nil && !tx1.prevtx2.validity[validator] {
			if _, exists := ds[*tx1.prevtx2]; !exists {
				if _, exists := count[tx1.prevtx2.count]; exists {
					//unconfirmed ds
					fmt.Printf("Unconfirmed DS2 detected can't confirm tip: %v %v %v \n", tx1.count, tx1.prevtx1.count, tx1.prevtx2.count)
					*result = false
				}
				//fmt.Printf("Tip2: %v %v \n", tx1.prevtx1.count, tx1.prevtx2.count)
				count[tx1.prevtx1.count] = struct{}{}
				ds[*tx1.prevtx1] = struct{}{}
				traverse(*tx1.prevtx2, validator, ds, count, result)
			}
		}
		reached = true
	}
}

//throw validator tx for validator n to confirm all tips
func validateTx(tips []*tx, v []vtx, validator int) []vtx {

	dsconf1 := make(map[tx]struct{})
	dsconf2 := make(map[int]*tx)

	for _, tip := range tips {
		//fmt.Printf("Validator: %v\n", validator)
		//fmt.Printf("tx: %v, confirmed: %v, references: %v,%v \n", tip.count, tip.confirm, tip.prevtx1.count, tip.prevtx2.count)
		// initialize for every tip
		dsTip := make(map[tx]struct{})
		count := make(map[int]struct{})

		result := true
		// don't reference both double spent in the same vtx unless an order for validator has already established
		traverse(*tip, validator, dsTip, count, &result)
		if result {
			v = append(v, vtx{&v[len(v)-1], tip, v[len(v)-1].count + 1})
			//confirm tip and tx refered by that tip
			confirm(tip, validator, dsconf1, dsconf2)
			//fmt.Printf("Choose tx: %v, confirmed: %v, references: %v,%v \n", tip.count, tip.validity, tip.prevtx1.count, tip.prevtx2.count)
		}
	}
	return v
}

//collect all tx referenced by vtx n
func checkDS(vtx1 []vtx, validator int) {
	//since vtx are in order we can determine the ds here
	//collect all tx referenced by vtx n
	collect := make(map[tx]struct{})
	ref := make(map[int]*tx)
	for _, svtx := range vtx1 {
		//collect all tx referenced by vtx n
		//fmt.Printf("Validator: %v, vtx: %v\n", validator, svtx.count)
		if svtx.tx != nil {
			collectTx(svtx.tx, collect, ref, validator)
		}
	}
}

func collectTx(tx *tx, collect map[tx]struct{}, ref map[int]*tx, validator int) {
	if _, exists := collect[*tx]; !exists {
		if _, exists := ref[tx.count]; exists {
			//ds only confirm oldest
			fmt.Printf("DS: %v\n", tx.count)
			tx.validity[validator] = false
		}
		collect[*tx] = struct{}{}
		ref[tx.count] = tx
	}
	reached := false
	for !reached {
		if tx.prevtx1 != nil {
			if _, exists := collect[*tx.prevtx1]; !exists {
				if _, exists := ref[tx.prevtx1.count]; exists {
					//ds only confirm oldest
					fmt.Printf("DS1: %v\n", tx.prevtx1.count)
					tx.prevtx1.validity[validator] = false
				}
				ref[tx.prevtx1.count] = tx.prevtx1
				collect[*tx.prevtx1] = struct{}{}
				collectTx(tx.prevtx1, collect, ref, validator)
			}
		}
		if tx.prevtx2 != nil {
			if _, exists := collect[*tx.prevtx2]; !exists {
				if _, exists := ref[tx.prevtx2.count]; exists {
					//ds only confirm oldest
					fmt.Printf("DS2: %v\n", tx.prevtx2.count)
					tx.prevtx2.validity[validator] = false
				}
				ref[tx.prevtx2.count] = tx.prevtx2
				collect[*tx.prevtx2] = struct{}{}
				collectTx(tx.prevtx2, collect, ref, validator)
			}
		}
		reached = true
	}
}

//color a node if its tx is valid (>2/3)
func confirmColor(tx *tx, node *dot.Node) {
	if tx.confirm {
		node.Set("color", "green")
	}
}

// simple testcase without double spent and all the validators confirm the same tips
// can change numTx
func dagtest1() ([]*tx, map[int][]vtx) {

	var txList []*tx
	//create initial tx that references nothing
	genesis := tx{nil, nil, [validators]bool{}, false, 0}
	//need two for references
	genesis2 := tx{&genesis, nil, [validators]bool{}, false, 1}
	//append
	txList = append(txList, &genesis)
	txList = append(txList, &genesis2)

	//create normal tx
	numTx := 4
	txList = newTx(numTx, txList)

	vList := make(map[int][]vtx)

	//tips to confirm
	tips := getTips(txList)

	//validate
	for i := 0; i < validators; i++ {
		//need to create first vtx first
		vList[i] = append(vList[i], vtx{nil, nil, 1})
		vList[i] = validateTx(tips, vList[i], i)
	}

	return txList, vList
}

// simple testcase without double spent and the validators confirm different tips
func dagtest2() ([]*tx, map[int][]vtx) {

	var txList []*tx
	//create initial tx that references nothing
	genesis := tx{nil, nil, [validators]bool{}, false, 0}
	//need two for references
	genesis2 := tx{&genesis, nil, [validators]bool{}, false, 1}
	//append
	txList = append(txList, &genesis)
	txList = append(txList, &genesis2)

	//create normal tx
	numTx := 3
	txList = newTx(numTx, txList)

	vList := make(map[int][]vtx)

	//tips to confirm
	tips := getTips(txList)

	var split int
	if validators%2 == 1 {
		split = validators/2 + 1
	}
	split = validators / 2

	//validate first half of validators
	for i := 0; i < split; i++ {
		//need to create first vtx first
		vList[i] = append(vList[i], vtx{nil, nil, 0})
		vList[i] = validateTx(tips, vList[i], i)
	}

	//create normal tx
	txList = newTx(numTx, txList)
	//tips to confirm
	tips = getTips(txList)

	//validate second half of validators
	for i := split; i < validators; i++ {
		//need to create first vtx first
		vList[i] = append(vList[i], vtx{nil, nil, 0})
		vList[i] = validateTx(tips, vList[i], i)
	}

	return txList, vList
}

// simple testcase without double spent and all the validators confirm the same tips
// can change numTx
func dagds1() ([]*tx, map[int][]vtx) {

	var txList []*tx
	//create initial tx that references nothing
	genesis := tx{nil, nil, [validators]bool{}, false, 0}
	//need two for references
	genesis2 := tx{&genesis, nil, [validators]bool{}, false, 1}
	//append
	txList = append(txList, &genesis)
	txList = append(txList, &genesis2)

	//create normal tx
	numTx := 5
	txList = newTx(numTx, txList)

	//insert ds
	txList = insertDS(txList)

	//tips to confirm
	tips := getTips(txList)

	vList := make(map[int][]vtx)

	//validate
	for i := 0; i < validators; i++ {
		//need to create first vtx first
		vList[i] = append(vList[i], vtx{nil, nil, 1})
		vList[i] = validateTx(tips, vList[i], i)
		//check for ds
		checkDS(vList[i], i)
	}

	return txList, vList
}

// simple testcase without double spent and validators confirm different tips
// can change numTx
func dagds2() ([]*tx, map[int][]vtx) {

	var txList []*tx
	//create initial tx that references nothing
	genesis := tx{nil, nil, [validators]bool{}, false, 0}
	//need two for references
	genesis2 := tx{&genesis, nil, [validators]bool{}, false, 1}
	//append
	txList = append(txList, &genesis)
	txList = append(txList, &genesis2)

	//create normal tx
	numTx := 6
	txList = newTx(numTx, txList)

	//insert ds
	txList = insertDS(txList)

	//tips to confirm
	tips := getTips(txList)

	vList := make(map[int][]vtx)

	var split int
	if validators%2 == 1 {
		split = validators/2 + 1
	}
	split = validators / 2

	//validate
	for i := 0; i < split; i++ {
		//need to create first vtx first
		vList[i] = append(vList[i], vtx{nil, nil, 1})
		vList[i] = validateTx(tips, vList[i], i)
		//check for ds
		checkDS(vList[i], i)
	}

	//create normal tx
	numTx = 4
	txList = newTx(numTx, txList)

	//tips to confirm
	tips = getTips(txList)

	//validate
	for i := split; i < validators; i++ {
		//need to create first vtx first
		vList[i] = append(vList[i], vtx{nil, nil, 1})
		vList[i] = validateTx(tips, vList[i], i)
		//check for ds
		checkDS(vList[i], i)
	}

	//validate
	for i := 0; i < split; i++ {
		vList[i] = validateTx(tips, vList[i], i)
		//check for ds
		checkDS(vList[i], i)
	}

	return txList, vList
}

func main() {

	var txList []*tx

	vList := make(map[int][]vtx)

	args := os.Args[1:]

	if len(args) == 2 {
		if args[1] == "ran" {
			ranDag = true
		}
	}

	if len(args) > 0 {
		switch arg := args[0]; arg {
		case "1":
			txList, vList = dagtest1()
		case "2":
			txList, vList = dagtest2()
		case "3":
			txList, vList = dagds1()
		case "4":
			txList, vList = dagds2()
		}
	} else {
		//default
		txList, vList = dagtest1()
	}

	//check confirm status for tx
	for i := range txList {
		//iterate all validators
		rThreshold := 0
		for j := range txList[i].validity {
			if txList[i].validity[j] == true {
				rThreshold++
			}
			if rThreshold >= threshold {
				// confirm tx that reached threshold
				txList[i].confirm = true
			}
		}
	}

	//make a dot file for graphviz
	graph := dot.NewGraph("G")
	graph.SetType(dot.DIGRAPH)

	//print txList

	//check for double spent and print them accordingly
	ds := make(map[int]struct{})
	dsList := make(map[*tx]struct{})
	for _, tx := range txList {
		//genesis
		if tx.count == 0 {
			node := dot.NewNode("0")
			confirmColor(tx, node)
			graph.AddNode(node)
		}
		//tx that references genesis only
		if tx.count == 1 {
			node := dot.NewNode("1")
			confirmColor(tx, node)
			graph.AddNode(node)
			edge := dot.NewEdge(dot.NewNode("0"), node)
			edge.Set("dir", "back") // edge backwards
			graph.AddEdge(edge)
		}
		//all other tx
		if tx.count > 1 {
			if _, exists := ds[tx.count]; !exists {
				ds[tx.count] = struct{}{}
				node := dot.NewNode(strconv.Itoa(tx.count))
				confirmColor(tx, node)

				strtx1 := strconv.Itoa(tx.prevtx1.count)
				strtx2 := strconv.Itoa(tx.prevtx2.count)
				// ref double-spent?
				for _, d1 := range dsG {
					// other ds tx?
					if d1.count == tx.count {
						node.Set("fontcolor", "red")
					}
					if tx.prevtx1 == d1 {
						strtx1 = strconv.Itoa(tx.prevtx1.count) + "ds"
					}
					if tx.prevtx2 == d1 {
						strtx1 = strconv.Itoa(tx.prevtx2.count) + "ds"
					}
				}
				graph.AddNode(node)

				edge1 := dot.NewEdge(dot.NewNode(strtx1), node)
				edge1.Set("dir", "back") // edge backwards
				edge2 := dot.NewEdge(dot.NewNode(strtx2), node)
				edge2.Set("dir", "back") // edge backwards
				graph.AddEdge(edge1)
				graph.AddEdge(edge2)
			} else {
				//handle double spent here
				dsList[tx] = struct{}{}
				node := dot.NewNode(strconv.Itoa(tx.count) + "ds")
				confirmColor(tx, node)
				node.Set("fontcolor", "red")
				graph.AddNode(node)
				edge1 := dot.NewEdge(dot.NewNode(strconv.Itoa(tx.prevtx1.count)), node)
				edge1.Set("dir", "back") // edge backwards
				edge2 := dot.NewEdge(dot.NewNode(strconv.Itoa(tx.prevtx2.count)), node)
				edge2.Set("dir", "back") // edge backwards
				graph.AddEdge(edge1)
				graph.AddEdge(edge2)
			}
			//fmt.Printf("tx: %v, confirmed: %v, references: %v,%v validity: %v \n", tx.count, tx.confirm, tx.prevtx1.count, tx.prevtx2.count, tx.validity)
		}

	}

	//print vtx
	for i := 0; i < validators; i++ {
		//fmt.Printf("Validator %v \n", i)
		for j, vtx := range vList[i] {
			if j == 0 {
				node := dot.NewNode("V" + strconv.Itoa(i) + "=" + strconv.Itoa(vtx.count))
				node.Set("color", "blue") //green,...,green4
				graph.AddNode(node)
			}
			if j > 0 {
				node := dot.NewNode("V" + strconv.Itoa(i) + "=" + strconv.Itoa(vtx.count))
				node.Set("color", "blue") //green,...,green4
				graph.AddNode(node)
				//ref prev vtx
				//string for prev vtx
				prev := "V" + strconv.Itoa(i) + "=" + strconv.Itoa(vtx.prev.count)
				edge1 := dot.NewEdge(dot.NewNode(prev), node)
				edge1.Set("dir", "back") // edge backwards
				//ref normal tx
				if _, exists := dsList[vtx.tx]; !exists {
					edge2 := dot.NewEdge(dot.NewNode(strconv.Itoa(vtx.tx.count)), node)
					edge2.Set("dir", "back")   // edge backwards
					edge2.Set("color", "blue") // blue
					graph.AddEdge(edge1)
					graph.AddEdge(edge2)
				} else {
					//ds
					edge2 := dot.NewEdge(dot.NewNode(strconv.Itoa(vtx.tx.count)+"ds"), node)
					edge2.Set("dir", "back")  // edge backwards
					edge2.Set("color", "red") // blue
					graph.AddEdge(edge1)
					graph.AddEdge(edge2)
				}
				//fmt.Printf("vtx: %v, prev: %v, tx: %v \n", vtx.count, vtx.prev.count, vtx.tx.count)
			}
		}
	}

	file, err := os.Create("g.dot")
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	fmt.Fprint(file, graph)

}
