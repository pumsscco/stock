package main
//新股只数统计，包括可转债在内，分析历年打新中签数量
import (
	"time"
	"fmt"
	"encoding/json"
)
type Cnt struct {
	Year int 			//年份
	Main, CB int 		//前者是主板，后者是可转债
}
func getNewShareCnt() (cnts []Cnt) {
	//先尝试从redis中抓取打新股票的只数统计列表，如果不成，再查数据库！
	key:=fmt.Sprintf("stock:new-share:count")
	val, err := client.Get(key).Result()
	if err == nil {
		json.Unmarshal([]byte(val),&cnts)
		return
	} else {
		errinfo:=fmt.Sprintf("get new share count list from redis error: %s" ,err)
		logger.Println(errinfo)
	}
	var minYear,maxYear int
	mainMap,cbMap:=make(map[int]int),make(map[int]int)
	//先列出最初与最新年份
	dealYearSQL:="select year(min(date)),year(max(date)) from stock"
	Db.QueryRow(dealYearSQL).Scan(&minYear,&maxYear)
	//再分别查两种新股的中签只数
	mainMapSQL:="SELECT year(date) y ,count(id) FROM `stock` where operation in ('申购中签','配股缴款') and code regexp '^[0,3,6,8,9]' group by y"
	rowsM, _ := Db.Query(mainMapSQL)
	for rowsM.Next() {
		var tmpY,tmpC int
		rowsM.Scan(&tmpY,&tmpC)
		mainMap[tmpY]=tmpC
	}
	cbMapSQL:="SELECT year(date) y ,count(id) FROM `stock` where operation in ('申购中签','配股缴款') and code regexp '^1' group by y"
	rowsC, _ := Db.Query(cbMapSQL)
	for rowsC.Next() {
		var tmpY,tmpC int
		rowsC.Scan(&tmpY,&tmpC)
		cbMap[tmpY]=tmpC
	}
	for i := maxYear; i >=minYear ; i-- {
		var cnt Cnt
		cnt.Year=i
		m, ok := mainMap[i]
		if !ok {
			cnt.Main=0
		} else {
			cnt.Main=m
		}
		c, ok := cbMap[i]
		if !ok {
			cnt.CB=0
		} else {
			cnt.CB=c
		}
		if cnt.CB!=0 || cnt.Main!=0 {
			cnts = append(cnts, cnt)
		}
	}
	s,err:=json.Marshal(cnts)
    if err!=nil {
		errinfo:=fmt.Sprintf("set new share count list to redis error: %s",err)
        logger.Println(errinfo)
    } else {
        client.Set(key, string(s), 75*time.Hour)
    }
	return
}
