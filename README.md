# bolt-nut db
A minimalistic wrapper for the [bbolt](https://github.com/etcd-io/bbolt) package, that was made just to work.


### Example file
In this example, it is shown hot to:
* Setup your database
* Insert items to the database
* Update items in the database
* Iterate through Buckets 
``` go
package main

import (
	"fmt"

	boltnut "github.com/appleGun22/bolt-nut"
)

// Important to CAPITALIZE the first letter of structures and its fields !!!
type Banana struct {
	Owner string
	Ripe  bool
}

type Monkey struct {
	Kind string
	Age  uint
}

func setup_db() *boltnut.DB {
	var buckets []string

	buckets = append(buckets, "monkeys")
	buckets = append(buckets, "bananas")

	path := "boltnut.db"

	db, e := boltnut.Init(path, &buckets)

	if e != nil {
		panic(e)
	}

	return db
}

func main() {
	db := setup_db()

	/* --- INSERT --- */

	// We create a WriteTx transaction, that means we want to modify the database
	// functions can be created directly inside the transaction, or you can pass one
	e := db.WriteTx(func(tx *boltnut.TX) error {
		// Add monkeys to the database
		// To do that first we access the "monkeys" bucket, that is filled with `Monkey` values
		monkeys := boltnut.Bucket[Monkey](tx, "monkeys")

		// Generate some monkeys
		monkey_team := make(map[string]Monkey, 3)
		monkey_team["moka"] = Monkey{"Gorilla", 1}
		monkey_team["pat"] = Monkey{"Gibbon", 3}
		monkey_team["robert"] = Monkey{"Macaque", 3}

		// monkeys name is our key
		for k, v := range monkey_team {
			// Now we Insert the monkeys into the database, if any monkey from our team exists it gets overwritten
			// Insert may return an error, we can handle it or return it and break out of the transaction
			e := monkeys.Insert([]byte(k), &v)

			if e != nil {
				return e
			}
		}

		// Add bananas to the database
		// This time we access the "bananas" bucket, that is filled with `Banana` values
		bananas := boltnut.Bucket[Banana](tx, "bananas")

		// Generate some bananas
		banana_box := make([]Banana, 5)
		banana_box[0] = Banana{"moka", false}
		banana_box[1] = Banana{"moka", true}
		banana_box[2] = Banana{"moka", false}
		banana_box[3] = Banana{"pat", false}
		banana_box[4] = Banana{"robert", false}

		// index used as key
		for k, v := range banana_box {
			// Because integers are not directly convertible from []byte()
			// we use this boltnut function to turn the integer into bytes
			e := bananas.Insert(boltnut.Itob(k), &v)

			if e != nil {
				return e
			}
		}

		return nil
	})

	// If our transaction returned an error
	if e != nil {
		panic(e)
	}

	/* --- UPDATE --- */

	// We want all of our bananas to get ripe, so we jump by one year forward
	// (dont wait a year for your bananas to get ripe :) )
	e = db.WriteTx(func(tx *boltnut.TX) error {

		bananas := boltnut.Bucket[Banana](tx, "bananas")

		// We previously created N bananas , we will iterate until not found
		for i := 0; ; i++ {
			var v Banana
			k := boltnut.Itob(i)

			// Get the Value
			e := bananas.Get(k, &v)
			if e == boltnut.ErrKeyNotFound {
				break
			} else if e != nil {
				return e
			}

			// Update what we need
			v.Ripe = true

			// Insert it back to the databse
			e = bananas.Insert(k, &v)
			if e != nil {
				return e
			}
		}

		// Monkeys got older by a year
		monkeys := boltnut.Bucket[Monkey](tx, "monkeys")

		monkey_names := []string{"moka", "pat", "robert"}
		for i := range monkey_names {
			var v Monkey
			k := []byte(monkey_names[i])

			e := monkeys.Get(k, &v)
			if e != nil {
				return e
			}

			v.Age += 1

			e = monkeys.Insert(k, &v)
			if e != nil {
				return e
			}
		}

		return nil
	})

	// If our transaction returned an error
	if e != nil {
		panic(e)
	}

	/* --- ForEach --- */

	e = db.WriteTx(func(tx *boltnut.TX) error {

		bananas := boltnut.Bucket[Banana](tx, "bananas")

		leaderboard := make(map[string]int)

		// For each Banana in bananas bucket, we perform this function
		e := bananas.ForEach(func(k []byte, v *Banana) error {

			leaderboard[v.Owner] += 1

			return nil
		})

		if e != nil {
			return e
		}

		for name, amount := range leaderboard {
			fmt.Printf("%s has %d bananas\n", name, amount)
		}

		return nil
	})

	// If our transaction returned an error
	if e != nil {
		panic(e)
	}
}

```
