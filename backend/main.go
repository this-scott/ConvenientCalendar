package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

type rbTree struct {
	root *Node
}

// black is 0(false) red is 1(true)
// left and right are the node's children
// key is date and value is date's line address in bulk
type Node struct {
	key         int
	left, right *Node
	parent      *Node
	color       bool
	value       int
}

/*type CalEvent {
	Title string
	StartDate
}*/
// add a node to the tree with a key and a value
func (t *rbTree) insertNew(nkey int, nvalue int) {
	//creates a node and inserts it depending on if it's root or not
	node := &Node{key: nkey, value: nvalue, color: true}
	if t.root == nil {
		t.root = node
	} else {
		t.insertNode(t.root, node)
	}
	t.fixViolations(node)
}

// traverse 1 rung of ladder
func (t *rbTree) insertNode(root, node *Node) {
	//traverse left if node is less than root & vice versa
	if node.key < root.key {
		if root.left == nil {
			root.left = node
			node.parent = root
		} else {
			t.insertNode(root.left, node)
		}
	} else {
		if root.right == nil {
			root.right = node
			node.parent = root
		} else {
			t.insertNode(root.right, node)
		}
	}
}

func (t *rbTree) fixViolations(node *Node) {
	for node.parent != nil && node.parent.color == true {
		//if the node's parent is the grandparent's left child, call right child uncle
		//else (because the node's parent is the right child or doesn't exist) call granpa's left child uncle
		if node.parent == node.parent.parent.left {
			uncle := node.parent.parent.right
			//if uncle is red then set parent & uncle black & grandparent red. Then point to grandparent
			//else: is black then leftrotate parent if it is right child, and set parent to black and grandparent to true and rightrotate grandparent
			if uncle != nil && uncle.color == true {
				node.parent.color = false
				uncle.color = false
				node.parent.parent.color = true
				node = node.parent.parent
			} else {
				if node == node.parent.right {
					node = node.parent
					t.leftRotate(node)
				}
				node.parent.color = false
				node.parent.parent.color = true
				t.rightRotate(node.parent.parent)
			}
		} else {
			uncle := node.parent.parent.left
			//if uncle is red set parent & uncle to black and gpa red, then set current node to gpa
			//else (uncle is black)
			if uncle != nil && uncle.color == true {
				node.parent.color = false
				uncle.color = false
				node.parent.parent.color = true
				node = node.parent.parent
			} else {
				if node == node.parent.left {
					node = node.parent
					t.rightRotate(node)
				}
				node.parent.color = false
				node.parent.parent.color = true
				t.leftRotate(node.parent.parent)
			}
		}
	}
	t.root.color = false
}

func (t *rbTree) leftRotate(x *Node) {
	y := x.right
	x.right = y.left
	if y.left != nil {
		y.left.parent = x
	}
	y.parent = x.parent
	if x.parent == nil {
		t.root = y
	} else if x == x.parent.left {
		x.parent.left = y
	} else {
		x.parent.right = y
	}
	y.left = x
	x.parent = y
}

func (t *rbTree) rightRotate(y *Node) {
	x := y.left
	y.left = x.right
	if x.right != nil {
		x.right.parent = y
	}
	x.parent = y.parent
	if y.parent == nil {
		t.root = x
	} else if y == y.parent.left {
		y.parent.left = x
	} else {
		y.parent.right = x
	}
	x.right = y
	y.parent = x
}

func (t *rbTree) PrintInOrder() {
	t.printInOrder(t.root)
	fmt.Println()
}

func (t *rbTree) printInOrder(node *Node) {
	if node == nil {
		return
	}
	t.printInOrder(node.left)
	fmt.Printf("%d ", node.key)
	t.printInOrder(node.right)
}

func main() {
	//getting linecount on open because server will constantly be using it
	//create rbtrees  with event dates as keys and line locations as values
	file, err := os.OpenFile("bulk.ics", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		fmt.Printf("Fail to open file: %v", err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	eventIndex := rbTree{}
	lineIndex := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		if strings.Contains(line, "DTSTART;TZID") {
			//DONE: fix this to add last date of year into key
			tdate, err := strconv.Atoi(line[32:38])
			fmt.Println(tdate)
			if err != nil {
				fmt.Println("Error converting string to integer:", err)
				return
			}
			eventIndex.insertNew(tdate, lineIndex)
		}
		lineIndex++
	}

	//request handlers(1 endpoint per call)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, world")
	})

	//handle cal request
	http.HandleFunc("/cal", func(w http.ResponseWriter, r *http.Request) {
		//read calendar
		http.ServeFile(w, r, "bulk.ics")
	})

	//handle calendar insertion
	//will prob have to rewrite later
	http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Unauthorized")
			return
		}
		r.ParseForm()

		for key, value := range r.Form {
			fmt.Printf("%s = %s\n", key, value)

		}

		//TODO: append string to end of file containing new event information sent in form. Will need to fix "openFile" also need to remove END:VCALENDAR at bottom of file
		/*fsize, err := file.Stat()
		if err != nil {
			fmt.Println(err)
		}*/
		//removes END:VCALENDAR
		newSize, err := file.Seek(-13, io.SeekEnd)
		if err != nil {
			fmt.Println("Error seeking file:", err)
			return
		}
		if err := file.Truncate(newSize); err != nil {
			fmt.Println("Error truncating file:", err)
			return
		}

		if _, err := file.WriteString("BEGIN:VEVENT\nSUMMARY:" + r.FormValue("title") + "\nDTSTART;TZID=America/New_York:" + r.FormValue("year") + r.FormValue("date") + "T" + r.FormValue("time") +
			"\nDTEND;TZID=America/New_York:" + r.FormValue("eyear") + r.FormValue("edate") + "T" + r.FormValue("etime")); err != nil {
			fmt.Println(err)
		}

		if r.FormValue("description") != "" {
			if _, err := file.WriteString("\nDESCRIPTION:" + r.FormValue("Description")); err != nil {
				fmt.Println(err)
			}
		}

		if r.FormValue("location") != "" {
			if _, err := file.WriteString("\nLOCATION:" + r.FormValue("location")); err != nil {
				fmt.Println(err)
			}
		}

		//recurse info
		if r.FormValue("frequency") != "" {
			if _, err := file.WriteString("\nRRULE:FREQ=" + r.FormValue("frequency") + ";UNTIL=" + r.FormValue("udate") + "T" + r.FormValue("utime")); err != nil {
				fmt.Println(err)
			}
			if r.FormValue("frequency") == "MONTHLY" || r.FormValue("frequency") == "WEEKLY" {
				file.WriteString(";INTERVAL=" + r.FormValue("interval"))
				if r.FormValue("frequency") == "WEEKLY" {
					file.WriteString(";BYDAY=" + r.FormValue("days"))
				}
			}
		}

		if _, err := file.WriteString("\nEND:VCALENDAR"); err != nil {
			fmt.Println(err)
		}
	})

	//Setup listener for exit call. Use to close ics safely when server closes
	//TODO: Wrap this in a safeshutdown request
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	http.ListenAndServe(":8080", nil)

	<-sigCh
	err = file.Close()
	if err != nil {
		fmt.Println(err)
	}
}
