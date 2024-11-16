package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"net/http"
	"time"
)
//生成页面
func generateHTML(w http.ResponseWriter, data interface{}, filenames ...string) {
	var files []string
	for _, file := range filenames {
		files = append(files, fmt.Sprintf("templates/%s.html", file))
	}
	funcMap := template.FuncMap{ 
		"fdate": func(t time.Time) string { return t.Format("2006-01-02") },
	}
	t,_:=template.New("list.html").Funcs(funcMap).ParseFiles(files...)
	t.ExecuteTemplate(w, "layout", data)
}

//首页
func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	generateHTML(w, nil, "layout", "navbar", "index")
}

//交易记录
func dealList(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if r.Method=="GET" {
			names:=getNameMapOld("")
			generateHTML(w, &names, "layout", "navbar", "name-list")
		} else if r.Method=="POST" {
			deals := getDealList(r.PostFormValue("code"))
			generateHTML(w, &deals, "layout", "navbar", "deal-list")
		}
}
//持仓股票最新买卖记录
func holdLastDeal(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	sums := getHoldLastDeal()
	generateHTML(w, &sums, "layout", "navbar", "deal-list")
}
//清仓记录
func clearance(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	clears := getClearance()
	generateHTML(w, &clears, "layout", "navbar", "clearance")
}
//持仓统计
func position(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	positions := getPosition()
	generateHTML(w, &positions, "layout", "navbar", "position")
}
//新增交易记录
func newDeal(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if r.Method=="GET" {
		generateHTML(w, nil, "layout", "navbar", "add")
	} else if r.Method=="POST" {
		createDeal(w,r)
	}
}
//打新交易记录
func newStock(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	//获得打新类别的子类
	t:=ps.ByName("Type")
	if r.Method=="GET" {
		generateHTML(w, t, "layout", "navbar", "sort-ns")
	// 按排行方案罗列统计结果
	} else if r.Method=="POST" {
		//获得排行方案
		sortMethod:=r.PostFormValue("kind")
		//logger.Println("sort method: ", sortMethod)
		deals := getNewShareStats(t, sortMethod)
		if t=="cb" {
			generateHTML(w, &deals, "layout", "navbar", "ns/cb")
		} else if t=="main" {
			generateHTML(w, &deals, "layout", "navbar", "ns/main")
		}
		
	}
}

