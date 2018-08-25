package main

import (
	"flag"
	"time"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"crgParser/model"
	"regexp"
	"strings"
	"os"
	"encoding/csv"
)

var dateStart, dateEnd time.Time
var maker *string
var writer *csv.Writer
func main(){
	file, err := os.Create("result.csv")
	if err!=nil{
		fmt.Println("Error create out file!")
	}
	defer file.Close()
	writer = csv.NewWriter(file)
	writer.Comma='|';
	defer writer.Flush()

	maker=flag.String("make", "Toyota", "Make of car")
	dateStartS:=flag.String("sd", time.Now().Format("2006.01.02"), "Start date YYYY.MM.DD")
	dateEndS:=flag.String("ed", time.Now().Format("2006.01.02"), "End date YYYY.MM.DD")

	flag.Parse()

	layout := "2006.01.02"
	dateStart, err=time.Parse(layout, *dateStartS)
	if err != nil {
		fmt.Println(err)
	}

	dateEnd, err=time.Parse(layout, *dateEndS)
	if err != nil {
		fmt.Println(err)
	}

	if dateStart.Month()!=time.August || dateEnd.Month()!=time.August{
		return
	}

	fmt.Println(*maker)
	fmt.Println(dateStart.Format("2006.01.02"))
	fmt.Println(dateEnd.Format("2006.01.02"))

	cityes:=getCites()
	for _, city := range cityes {
		getPagesCars(city)
	}
	
}


func getCites() []model.City{
	var cityes []model.City
	doc,err := goquery.NewDocument("https://www.craigslist.org/about/sites")
	if err != nil {
		panic(err)
	}
	us:=doc.Find(".colmask").First()
	us.Find("a").Each(func(i int, selection *goquery.Selection) {
		p, exist:= selection.Attr("href")
		if (exist){
			cityes=append(cityes, model.City{selection.Text(), p})
		}

	})
	return cityes
}

func getPagesCars(city model.City){
	fmt.Println("City: "+city.Name)
	doc,err := goquery.NewDocument(city.Path)

	if err != nil {
		panic(err)
	}
	cta, exist:=doc.Find(".cta").First().Attr("href")
	if exist {
		cityCta:=city.Path[:len(city.Path)-1]+cta
		getCarsByDate(cityCta+"?auto_make_model="+*maker, cityCta, city.Name)

	}
}

func getCarsByDate(path, basePath, city string){
	fmt.Println("Job: "+path)
	layout := "2006-01-02"
	docCity,err := goquery.NewDocument(path)
	if err != nil {
		panic(err)
	}
	badRange:=false
	docCity.Find(".result-info").EachWithBreak(func(i int, rowCarDoc *goquery.Selection) bool {
		nearby:=rowCarDoc.Find("nearby").First()
		if len(nearby.Nodes)!=0{
			return false
		}
		d, exist:=rowCarDoc.Find(".result-date").First().Attr("datetime")
		if exist{
			dateCta, err:=time.Parse(layout, d[:10])
			if err!=nil{
				fmt.Errorf("Error parse date: %s", err)
				return true
			}
			if dateCta.After(dateEnd){
				return true
			}else if dateCta.Before(dateStart) {
				badRange=true
				return false
			}
			a:=rowCarDoc.Find("a")
			pathCar, exist:=a.Attr("href")
			if exist{
				car:=new(model.Car)
				id, exist:=a.Attr("data-id")
				if exist{
					car.Id=id
					car.Make=*maker
					car.City=city
					getRecord(pathCar, *car)
				}else{
					fmt.Println("ID not found!")
					return true
				}

			}


		}
		return true
	})
	if (!badRange){
		pathCar, exist:=docCity.Find(".next").First().Attr("href")
		if exist && pathCar!="" {
			getCarsByDate(basePath[:len(basePath)-1]+pathCar, basePath, city)
		}
	}

}

func getRecord(path string, car model.Car)  {
	carPage,err := goquery.NewDocument(path)
	if err != nil {
		panic(err)
	}
	re := regexp.MustCompile("[0-9]+")
	attr:=carPage.Find(".attrgroup").First().Find("b").Text()
	if attr!="" {
		y:=re.FindAllString(attr, 1)
		if len(y)>0{
			car.Year=y[0]
		}
		s:=strings.Split(strings.ToLower(attr), strings.ToLower(*maker))
		if len(s)>1 {
			car.Model=strings.Title(strings.TrimSpace(s[1]))
		}
	}
	first:=true
	carPage.Find(".thumb").Each(func(i int, img *goquery.Selection) {
		imgPath, exist:=img.Attr("href")
		if exist{
			if first{
				car.Images=imgPath
				first=false
			}else{
				car.Images=car.Images+", "+imgPath
			}

		}
	})
	if first{
		imgPath, exist:=carPage.Find(".slide").Find("img").First().Attr("src")
		if exist{
			car.Images=imgPath
		}
	}
	record:=make([]string, 6)
	record[0]=car.Id
	record[1]=car.City
	record[2]=car.Make
	record[3]=car.Model
	record[4]=car.Year
	record[5]=car.Images
	writer.Write(record)
}