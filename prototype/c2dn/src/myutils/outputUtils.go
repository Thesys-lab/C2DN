package myutils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func DumpIntSlice(is []int64, filename string) {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		if _, err := os.Create(filename); err != nil {
			SugarLogger.Fatal(err)
		}
	}

	file, err := os.OpenFile(filename, os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		SugarLogger.Fatal(err)
		return
	}
	defer file.Close()

	for _, e := range is {
		if _, err := file.WriteString(strconv.FormatInt(e, 10) + "\n"); err != nil {
			fmt.Println("Error writing to file", filename, err.Error())
			SugarLogger.Fatal("Error writing to file", filename, err.Error())
			return
		}
	}
}

func DumpFloatSlice(fs []float64, filename string) {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		if _, err := os.Create(filename); err != nil {
			SugarLogger.Fatal(err)
		}
	}

	file, err := os.OpenFile(filename, os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		SugarLogger.Fatal(err)
		return
	}
	defer file.Close()

	for _, e := range fs {
		if _, err := file.WriteString(strconv.FormatFloat(e, 'f', 4, 64) + "\n"); err != nil {
			fmt.Println("Error writing to file", filename, err.Error())
			SugarLogger.Fatal("Error writing to file", filename, err.Error())
			return
		}
	}
}

func DumpFloatChan(fc chan float64, filename string, wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	_ = os.Remove(filename)
	if _, err := os.Create(filename); err != nil {
		SugarLogger.Fatal(err)
	}

	file, err := os.OpenFile(filename, os.O_WRONLY, 0644)
	writer := bufio.NewWriter(file)
	if err != nil {
		fmt.Println(err)
		SugarLogger.Fatal(err)
		return
	}
	defer file.Close()
	//SugarLogger.Debugf("dump float chan started for " + filename);

	for {
		e, ok := <-fc
		if !ok {
			break
		}
		t := time.Now().Unix()
		s := fmt.Sprintf("%v %v\n", t, e)
		//if _, err := writer.WriteString(strconv.FormatFloat(e, 'f', 4, 64) + "\n"); err != nil {
		if _, err := writer.WriteString(s); err != nil {
			SugarLogger.Fatal("Error writing to file", filename, err.Error())
		}

		if t%10 == 0 {
			_ = writer.Flush()
			_ = file.Sync()
		}
	}

	s := fmt.Sprintf("# %v %v\n", time.Now().Unix(), "end of output")
	if _, err := writer.WriteString(s); err != nil {
		SugarLogger.Fatal("Error writing to file", filename, err.Error())
	}

	_ = writer.Flush()
	_ = file.Sync()
	file.Close()
	SugarLogger.Debugf("dumpFloatChan done for " + filename)
}

func DumpChan(c chan interface{}, filename string, wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	_ = os.Remove(filename)
	if _, err := os.Create(filename); err != nil {
		SugarLogger.Fatal(err)
	}

	file, err := os.OpenFile(filename, os.O_WRONLY, 0644)
	writer := bufio.NewWriter(file)
	if err != nil {
		fmt.Println(err)
		SugarLogger.Fatal(err)
		return
	}
	defer file.Close()
	SugarLogger.Debugf("dump chan started for " + filename)

	for {
		e, ok := <-c
		if !ok {
			break
		}
		t := time.Now().Unix()
		s := fmt.Sprintf("%v %v\n", t, e)
		if _, err = writer.WriteString(s); err != nil {
			SugarLogger.Fatal("Error writing to file", filename, err)
		}
		if t%10 == 0 {
			_ = writer.Flush()
			_ = file.Sync()
		}
	}

	s := fmt.Sprintf("# %v %v\n", time.Now().Unix(), "end of output")
	if _, err := writer.WriteString(s); err != nil {
		SugarLogger.Fatal("Error writing to file", filename, err.Error())
	}

	_ = writer.Flush()
	_ = file.Sync()
	file.Close()
	SugarLogger.Info("dumpChan finished for " + filename)
}

func DumpThrpt(nClient int, throughput []int64, filename string, wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}
	//_, err := os.Stat(filename)
	//if os.IsNotExist(err) {

	_ = os.Remove(filename)
	if _, err := os.Create(filename); err != nil {
		SugarLogger.Fatal(err)
	}
	//}

	file, err := os.OpenFile(filename, os.O_WRONLY, 0644)
	writer := bufio.NewWriter(file)
	if err != nil {
		fmt.Println(err)
		SugarLogger.Fatal(err)
		return
	}
	defer file.Close()

	var totalThrpt int64 = 0
	for {
		for time.Now().Unix()%20 != 0 {
			time.Sleep(200 * time.Millisecond)
		}
		totalThrpt = 0
		for i := 0; i < nClient; i++ {
			totalThrpt += atomic.SwapInt64(&throughput[i], 0)
		}
		if totalThrpt < 0 {
			// all finish
			break
		}

		if _, err := writer.WriteString(strconv.FormatInt(time.Now().Unix(), 10) + ":" +
			strconv.FormatFloat(float64(totalThrpt)/float64(1000000.0), 'f', 4, 64) + "\n"); err != nil {
			SugarLogger.Fatal("Error writing to file", filename, err.Error())
		}
		_ = writer.Flush()
		time.Sleep(19 * time.Second)
	}
	SugarLogger.Debugf("throughput output has finished")
}
