package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	. "strings"
)


type URLs struct {
	URLs []URL `json:"URLs"` // массив структур URL
}

type URL struct { // структура из элементов JSON-файла, содержащая поля:
	ID int `json:"id"` // идентификатора
	Address string `json:"address"` // адреса исходного
	Short string `json:"short"` // адреса зашифрованного
}

// функция реверса массив для алгоритма base62
func reverse (arr[]int) []int {
	var reverseString []int
	for i := len(arr) - 1; i >= 0; i--  {
		reverseString = append(reverseString, arr[i])
	}
	return reverseString
}

// Алгоритм base62
// имеем словарь,  в котором представлено 62 ключа и, значит, 62 элемента
// каждый элемент - это символ из последовательности [a..z, A..Z, 0..9]
// каждый ключ - это число от 0 до 62
// при занесении мы берём следующий идентификатор из JSON
// после последнего в предыдущей БД
// Например, последний в БД сейчас - 54
// значит, добавится 55
// После всего этого переводим чсило в 62-ичную систему счисления
// путём получения остатков от деления на 62 каждый раз
// реверсим полученный массив остатков
// и по ключам, соответствующим цислам в ревёрснутом массиве, находим буквы для кода
func base62 (code_table map[int]string, id int) string {
	var code []int
	for ; id > 0; {
		code = append(code, id%62)
		id /= 62
	}
	code = reverse(code)
	result := ""
	for _, i := range code {
		result += code_table[i]
	}
	return result // "http://avi.to/" + result
}

// чтение содержимого JSON-файла
func jsonRead () URLs{
	byteValue, _ := ioutil.ReadFile("URLs.json")
	var url URLs
	err := json.Unmarshal(byteValue, &url)
	if err != nil {
		panic("Unmarshall was bad")
	}
	return url
}

// запись БД в JSON-файл
func jsonWrite (url URLs) {
	decodeJson, err := json.MarshalIndent(url, "", "    ")
	if err != nil {
		fmt.Println(err)
		panic("There is an error!")
	}
	err = ioutil.WriteFile("URLs.json", decodeJson, 0644)
	return
}

// создаём карту для алгоритма base62
func makingBaseMap ()map[int]string {
	m := make(map[int]string)
	for i := 0; i < 51; i++ {
		if i < 26 {
			m[i] = string(i + 97) // до 25 - [a..z]
		} else {
			m[i] = string(i + 39) // далее до  51 - [A..Z]
		}
	}
	for i:= 51; i <= 61; i++ { // с 52 по 62 - [0..9]
		m[i] = string(i - 3)
	}
	return m
}

// добавление простой ссылки
func jsonAppend(url URLs, m map[int]string) URLs{
	var id int = url.URLs[len(url.URLs) - 1].ID + 1
	var add string
	fmt.Println("Enter your URL:")
	fmt.Fscan(os.Stdin, &add) // чтение исходной ссылки
	add = validateURL(add) // проверка на валидность и ниже bse62 с добавлением структуры в массив url.URLs
	newStruct := URL{ID: id, Address: add, Short: base62(m, id)}
	url.URLs = append(url.URLs, newStruct)
	return url
}

// добавление кастомной ссылки в БД
func jsonAppendCustom(url URLs) URLs{
	var id int = url.URLs[len(url.URLs) - 1].ID + 1
	var add string
	var custom string
	fmt.Println("Enter your URL:")
	fmt.Fscan(os.Stdin, &add) // читаем исходную ссылку
	add = validateURL(add) // проверяем валидность
	fmt.Println("Enter custom URL you want:")
	fmt.Fscan(os.Stdin, &custom) // чтение кастомного кода
	redirectingUrl := custom // "http://avi.to/" + custom
	newStruct := URL{ID: id, Address: add, Short: redirectingUrl}
	url.URLs = append(url.URLs, newStruct)
	return url
}

// проверка валидности введённой ссылки (которая исходная)
func validateURL (check string) string{
	u, err := url.Parse(check) // тут ссылку парсим
	if u.Scheme == "" { // тут приводим её к виду с http://
		check = "http://" + check
	}
	u, err = url.ParseRequestURI(check) // снова парсим, чтобы учесть http://
	// нижу мы проверяем наличие точки в ссылке как обязательно части
	// точка не в конце и не в начале, а в середине
	// хост тоже пустовать не должен. http:// или /src/myProject в ссылках не пройдут!
	if err != nil || u.Host == "" || ContainsAny(check, ".") == false || HasSuffix(u.Host, ".") == true || HasPrefix(u.Host, ".") == true {
		panic("URL does not exist!")
	}
	return check + "/"
}

// редирект-функция, достающая закодированную часть (Path) из адреса страницы
// и находящая его в БД, а затем перенаправлящая по найденной ссылке
func redirect(w http.ResponseWriter, r *http.Request) {
	redirecting := r.URL.Path[1:]
	needed := findURL(jsonRead(), redirecting)
	http.Redirect(w, r, needed, 301)
}

// поиск переданного закодированного URL-адреса в БД
func findURL (UrlSlice URLs, redir string) string {
	for _, i := range(UrlSlice.URLs){
		if Compare(i.Short, redir) == 0 {
			fmt.Println("Now you are entering: " + i.Address)
			return i.Address
		}
	}
	return ""
}

func main(){
	fmt.Println("What do you want to do?") // выбираем одно из трёх:
	fmt.Println("1. Decode your URL") // кодируем адрес base62-алгоритмом
 	fmt.Println("2. Create custom URL") // создаём кастомный URL сами
	fmt.Println("3. Redirect") // запускаем перенаправление по закодированным ссылкам
	var choose byte
	fmt.Scanf("%d", &choose)
	switch choose { // сам процесс выбора
	case 1:
		m := makingBaseMap() // создаём словарь для base62
		url := jsonRead() // читаем из JSON-файла
		url = jsonAppend(url, m) // добавляем закодированный URL
		jsonWrite(url) // записываем в JSON-файл БД с новым URL
	case 2:
		url := jsonRead() // читаем из JSON-файла содержимое
		url = jsonAppendCustom(url)
		jsonWrite(url) // записываем в JSON-файл БД с новым URL
	case 3:
		http.HandleFunc("/", redirect) // поднимаем сервер на порту 9090
		err := http.ListenAndServe(":9090", nil)
		if err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}
}