package main

type WeatherData struct {
	City        string  `json:"city"`
	Temperature float64 `json:"temperature"`
	WindSpeed   float64 `json:"wind_speed"`
}

type WeatherRequest struct {
	// The city for which to request the data
	City string `json:"city"`
	// Return windspeed in km/h
	WindSpeed bool `json:"wind_speed"`
	// Return temperature in Celsius
	Temperature bool `json:"temperature"`
}

func getWeather(request WeatherRequest) WeatherData {
	return WeatherData{
		City:        request.City,
		Temperature: 23.0,
		WindSpeed:   10.0,
	}
}

type WeatherOnDayRequest struct {
	WeatherRequest
	// The date for which to request the data
	Date string `json:"date"`
}

func getWeatherOnDay(request WeatherOnDayRequest) WeatherData {
	return WeatherData{
		City:        request.City,
		Temperature: 23.0,
		WindSpeed:   10.0,
	}
}
