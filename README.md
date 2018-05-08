# dBA for DAG 

## Overview

Proof of Concept for reaching consensus in DAG via dBA (dynamic Byzantine Agreement), which can withstand n >= 3f+1 byzantine failures.

## Requirements

* git
* go 1.9+

are required to run this.

## Installation

     $ go get github.com/AidosKuneen/dBA

## Usage

    $ go run main.go `1-4` `ran`

* `1,2` - Testcases without double spent
* `3,4` - Testcases with double spent
* `ran` - create random dag (if not the generated dag will be the same for each testcase)

Number of Validators can be changed via the `bft` variable (`bft` is the maximum number of faulty validators that are tolerable)

Running the programm generates a dot file `g.dot` in the current directory. `g.dot` is a graphviz file that can be viewed with xDot for example.