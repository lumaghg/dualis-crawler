package crawler

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	BaseURL = "https://dualis.dhbw.de"
)

type App struct {
	Client *http.Client
}

type LoginInput struct {
	//to be found in HTML
	Appname   string `json:"APPNAME"`
	Prgname   string `json:"PRGNAME"`
	Arguments string `json:"ARGUMENTS"`
	Clino     string `json:"clino"`
	Menuno    string `json:"menuno"`
	Menu_type string `json:"menu_type"`
	//to be retrieved from database
	Usrname   string `json:"usrname"`
	Pass      string `json:"pass"`
	Browser   string
	Plattform string
}

type Course struct {
	Name         string        `json:"name"`
	Examinations []examination `json:"examinations"`
}

type examination struct {
	Exam_type string `json:"exam_type"`
	Grade     string `json:"grade"`
}

func GetDualisCrawlResults(email string, password string) ([]Course, error) {
	jar, _ := cookiejar.New(nil)

	app := App{
		Client: &http.Client{Jar: jar},
	}

	loginInput, err := app.getLoginData()
	if err != nil {
		return []Course{}, err
	}
	refreshURL, err := app.performLoginAndGetRefreshURL(loginInput)
	if err != nil {
		log.Fatal(err)
		return []Course{}, err
	}
	gradePageURL, err := app.getGradePageURL(refreshURL)
	if err != nil {
		return []Course{}, err
	}

	gradeDetailLinks, err := app.extractGradeDetailLinks(gradePageURL)
	if err != nil {
		return []Course{}, err
	}
	courses, err := app.extractGrades(gradeDetailLinks)
	if err != nil {
		return []Course{}, err
	}
	return courses, nil
}

func (app *App) getLoginData() (LoginInput, error) {
	loginURL := BaseURL + "/scripts/mgrqispi.dll?APPNAME=CampusNet&PRGNAME=EXTERNALPAGES&ARGUMENTS=-N000000000000001,-N000324,-Awelcome"
	client := app.Client
	//get login Page document
	response, err := client.Get(loginURL)

	if err != nil {
		log.Fatalln("Error fetching response. ", err)
	}

	defer response.Body.Close()
	// convert response to Document
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		log.Fatal("Error loading HTTP response body. ", err)
	}
	//find hidden values
	appname, _ := document.Find("input[name='APPNAME']").Attr("value")
	prgname, _ := document.Find("input[name='PRGNAME']").Attr("value")
	arguments, _ := document.Find("input[name='ARGUMENTS']").Attr("value")
	clino, _ := document.Find("input[name='clino']").Attr("value")
	menuno, _ := document.Find("input[name='menuno']").Attr("value")
	menu_type, _ := document.Find("input[name='menu_type']").Attr("value")

	loginInput := LoginInput{
		Appname:   appname,
		Prgname:   prgname,
		Arguments: arguments,
		Clino:     clino,
		Menuno:    menuno,
		Menu_type: menu_type,
		Browser:   "",
		Plattform: "",
		Usrname:   "s201808@student.dhbw-mannheim.de",
		Pass:      "xj3ghgPUx",
	}

	return loginInput, err
}

func (app *App) performLoginAndGetRefreshURL(loginInput LoginInput) (string, error) {
	//send Form to login page
	client := app.Client

	loginURL := BaseURL + "/scripts/mgrqispi.dll"

	data := url.Values{
		"APPNAME":   {loginInput.Appname},
		"PRGNAME":   {loginInput.Prgname},
		"ARGUMENTS": {loginInput.Arguments},
		"clino":     {loginInput.Clino},
		"menuno":    {loginInput.Menuno},
		"menu_type": {loginInput.Menu_type},
		"browser":   {loginInput.Browser},
		"plattform": {loginInput.Plattform},
		"usrname":   {loginInput.Usrname},
		"pass":      {loginInput.Pass},
	}

	response, err := client.PostForm(loginURL, data)

	if err != nil {
		log.Fatalln(err)
		return "", err
	}

	defer response.Body.Close()

	//extract auth cookie
	cookieHeader := response.Header.Get("Set-Cookie")
	if len(cookieHeader) < 1 {
		return "", fmt.Errorf("login failed")
	}
	cookieValStartIndex := strings.Index(cookieHeader, "=") + 1
	cookieValEndIndex := strings.Index(cookieHeader, ";")
	cookieValue := cookieHeader[cookieValStartIndex:cookieValEndIndex]

	//add cnsc cookie to CookieJar
	var cookies []*http.Cookie

	firstCookie := &http.Cookie{
		Name:   "cnsc",
		Value:  cookieValue,
		Path:   "/",
		Domain: ".dualis.dhbw.de",
	}

	cookies = append(cookies, firstCookie)
	cookieURL, _ := url.Parse(BaseURL)
	client.Jar.SetCookies(cookieURL, cookies)

	//extract refreshURL
	refreshHeader := response.Header.Get("Refresh")
	if len(refreshHeader) < 1 {
		return "", fmt.Errorf("refresh URL not found")
	}
	URLStartIndex := strings.Index(refreshHeader, "=") + 1
	refreshURL := refreshHeader[URLStartIndex:]

	return refreshURL, nil
}

func (app *App) getGradePageURL(refreshURL string) (string, error) {

	client := app.Client
	//first Refresh
	response, err := client.Get(BaseURL + refreshURL)
	if err != nil {
		return "", err
	}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	bodyString := string(bodyBytes)
	//find the start of the URL in the refresh header (there is only one URL substring)
	URLStartIndex := strings.Index(bodyString, "URL=") + 4
	//get Index at the end of the refresh URL but start at the URL Start, so that the index is also the length
	URLLength := strings.Index(bodyString[URLStartIndex:], "\"")
	//get the substring from the URLStart to the URLStart + URL Length = URL End
	startPageURL := bodyString[URLStartIndex : URLStartIndex+URLLength]
	if err != nil {
		return "", err
	}

	startPageResponse, err := client.Get(BaseURL + startPageURL)
	if err != nil {
		return "", nil
	}
	defer response.Body.Close()
	//find the URL to "Prüfungsergebnisse"
	document, _ := goquery.NewDocumentFromReader(startPageResponse.Body)
	gradePageURL, exists := document.Find("a[class='depth_1 link000307 navLink ']").Attr("href")
	if !exists {
		return "", fmt.Errorf("Grade page url not found")
	}
	return gradePageURL, nil
}

func (app *App) extractGradeDetailLinks(gradePageURL string) ([]string, error) {
	client := app.Client
	//get baseGradePage and extract the semester-options
	response, err := client.Get(BaseURL + gradePageURL)
	if err != nil {
		return []string{}, err
	}
	document, _ := goquery.NewDocumentFromReader(response.Body)

	var semesterArguments = []string{}
	document.Find("select[id='semester']").Find("option").Each(func(i int, selection *goquery.Selection) {
		semesterArguments = append(semesterArguments, selection.AttrOr("value", ""))

	})

	var gradeDetailLinks = []string{}
	//for every semesterArgument, request the page
	for _, argument := range semesterArguments {
		response, err := client.Get(BaseURL + gradePageURL + "-N" + argument)
		if err != nil {
			return []string{}, err
		}
		//extract all the ResultDetail URLS from the page's javascript
		document, _ := goquery.NewDocumentFromReader(response.Body)
		document.Find("td[class='tbdata']").Each(func(i int, s *goquery.Selection) {
			javaScriptText := s.Find("script").Text()
			if len(javaScriptText) > 0 {
				URLStartIndex := strings.Index(javaScriptText, "dl_popUp(") + 10
				URLLength := strings.Index(javaScriptText[URLStartIndex:], "\"")
				gradeDetailLinks = append(gradeDetailLinks, javaScriptText[URLStartIndex:URLStartIndex+URLLength])
			}
		})

	}
	return gradeDetailLinks, nil
}

func (app *App) extractGrades(gradeDetailLinks []string) ([]Course, error) {
	client := app.Client
	courses := []Course{}
	for _, gradeURL := range gradeDetailLinks {
		response, err := client.Get(BaseURL + gradeURL)
		if err != nil {
			return []Course{}, nil
		}
		document, _ := goquery.NewDocumentFromReader(response.Body)
		courseName, _ := document.Find("h1").Html()
		course := Course{
			Name: courseName,
		}
		examinations := []examination{}
		document.Find("table:first-of-type").Find("tr").Each(func(i int, s *goquery.Selection) {
			examination := examination{}
			s.Find("td[class='tbdata']").Each(func(i int, s *goquery.Selection) {
				if i == 1 {
					examination.Exam_type = strings.TrimSpace(s.Text())
				}
				if i == 3 {
					examination.Grade = strings.TrimSpace(s.Text())
				}
			})
			if examination.Exam_type != "" {
				examinations = append(examinations, examination)
			}
		})
		course.Examinations = examinations
		courses = append(courses, course)
	}
	for _, v := range courses {
		fmt.Println(v.Name)
		fmt.Println(v.Examinations)
		fmt.Println("")
	}
	return courses, nil
}
