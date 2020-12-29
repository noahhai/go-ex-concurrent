

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"errors"
	"sync"

	"github.com/Jeffail/gabs"
)

type Coordinate struct {
	Latitude    float64
	Longitude   float64
	CurrentTemp float64
}

const weatherCityUrlTemplate string = "https://www.metaweather.com/api//location/search/?lattlong=%f,%f"
const weatherUrlTemplate string = "https://www.metaweather.com/api/location/%d/%d/%d/%d"
const cityUrls string = "https://public.opendatasoft.com/api/records/1.0/search/?dataset=1000-largest-us-cities-by-population-with-geographic-coordinates&facet=city&facet=state&sort=population&rows=100"

func main() {

	cityData, err := doGetRequest(cityUrls)
	if err != nil {
		panic(err)
	}

	cityDataParsed, _ := gabs.ParseJSON(cityData)
	cities, _ := cityDataParsed.Path("records").Children()
	cityCoordinates := [100]Coordinate{}
	var wg sync.WaitGroup
	for i, city := range cities {
		wg.Add(1)
		go func(i int, city *gabs.Container) {
			coord := city.Path("fields.coordinates").Data().([]interface{})
			cityCoordinates[i] = Coordinate{
				Latitude:  coord[0].(float64),
				Longitude: coord[1].(float64),
			}
			if err, temp := getCurrentTemperatureForCoordinates(cityCoordinates[i]); err == nil {
				cityCoordinates[i].CurrentTemp = temp
			}
			wg.Done()
		}(i, city)

	}

	averageTemp := float64(0.0)
	count := 0
	wg.Wait()
	for _, c := range cityCoordinates {
		if c.CurrentTemp != float64(0.0) {
			averageTemp += c.CurrentTemp
			count++
		}
	}

	fmt.Println("Average temperature is ", averageTemp/float64(count))

}

func getCurrentTemperatureForCoordinates(coord Coordinate) (error, float64) {
	weatherCityData, err := doGetRequest(fmt.Sprintf(weatherCityUrlTemplate, coord.Latitude, coord.Longitude))
	if err != nil {
		return err, float64(0.0)
	}

	weatherCitiesParsed, _ := gabs.ParseJSON(weatherCityData)
	weatherCityWoeids := weatherCitiesParsed.Path("woeid").Data().([]interface{})
	weatherURLFormatted := fmt.Sprintf(weatherUrlTemplate, int64(weatherCityWoeids[0].(float64)), time.Now().Year(),
		int(time.Now().Month()), time.Now().Day())
	weatherData, err := doGetRequest(weatherURLFormatted)
	if err != nil {
		panic(err)
	}
	weatherDataParsed, _ := gabs.ParseJSON(weatherData)
	temp, ok := weatherDataParsed.Path("the_temp").Data().([]interface{})[0].(float64)
	if ok {
		return nil, temp
	}
	return errors.New("result was not a float"), float64(0.0)
}

func doGetRequest(url string) ([]byte, error) {
	fmt.Printf("starting request %s\n", url)
	res, err := http.Get(url)
	fmt.Printf("fetched request %s\n", url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
