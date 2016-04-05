package aarhusboligventeliste

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gopkg.in/xmlpath.v2"

	"net/http/cookiejar"

	"golang.org/x/net/context"
	"golang.org/x/net/publicsuffix"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/urlfetch"
)

//Error handling is almost non existing here, because it is a quick hack for my self

const appartmentEntityKind = "Appartment"

func init() {
	http.HandleFunc("/fetchStatus", statusFetcher)
	http.HandleFunc("/status", statusHandler)
	http.Handle("/", http.FileServer(http.Dir("static/")))

	log.Print(http.ListenAndServe(":8080", nil))
}

func logAndWriteError(msg string, res http.ResponseWriter) {
	log.Printf(msg)
	res.WriteHeader(http.StatusInternalServerError)
	res.Write([]byte(msg))
}

type statusResponse struct {
	Median      int
    NumberOfAppartments int
	Appartments []Appartment
}

func statusHandler(res http.ResponseWriter, req *http.Request) {

	ctx := appengine.NewContext(req)

	count, err := datastore.NewQuery(appartmentEntityKind).Count(ctx)
	if err != nil {
		logAndWriteError(fmt.Sprintf("Error while getting count %v", err), res)
		return
	}

    //Assumption is that there are more than twice the number of appartments than we are displaying
	q := datastore.NewQuery(appartmentEntityKind).Order("CurrentRank").Limit(count/2)

	var appartments []Appartment
	if _, err := q.GetAll(ctx, &appartments); err != nil {
		logAndWriteError(fmt.Sprintf("Error while getting appartments %v", err), res)
		return
	}
    
    //Median might be off by one appartment but thats ok for me
    median := appartments[len(appartments)-1].CurrentRank;

	response := statusResponse{median, count, appartments[:15]}

	json, err := json.Marshal(response)
	if err != nil {
		logAndWriteError(fmt.Sprintf("Error while encoding json %v", err), res)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.Write(json)
}

func statusFetcher(res http.ResponseWriter, req *http.Request) {


	if req.Header.Get("X-Appengine-Cron") != "true" {
		logAndWriteError("Not allowed to call cron endpoint", res)
		return
	}

	ctx := appengine.NewContext(req)
    conf := GetConfig(ctx);
	time := time.Now()

	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}

	client := urlfetch.Client(ctx)
	client.Jar = jar

	waitListResponse, err := client.PostForm("https://aarhusbolig.dk/umbraco/surface/MemberSurface/HandleLogin",
		url.Values{"Kodeord": {conf.Password}, "Brugernummer": {conf.Username}, "redirect": {"/din-venteliste"}})

	if err != nil {
		logAndWriteError(fmt.Sprintf("Error while getting waiting list %v", err), res)
		return
	}
	if waitListResponse.StatusCode != 200 {
		logAndWriteError(fmt.Sprintf("Unexpected status code %v received when getting waitinglist. %v",
			waitListResponse.StatusCode, waitListResponse),
			res)
		return
	}

	rootNode, err := xmlpath.ParseHTML(waitListResponse.Body)
	if err != nil {
		logAndWriteError(fmt.Sprintf("Error while reading body %v", err), res)
		return
	}
	cardPath := xmlpath.MustCompile("//*[@class='hc']")

	log.Printf("Parsing cards")

	for cardIter := cardPath.Iter(rootNode); cardIter.Next(); {
		cardNode := cardIter.Node()
		id := constructID(cardNode)
		rank := getRank(cardNode)
        if rank == 0 {
            logAndWriteError("Error getting rank, check log", res)
            return
        }
		persistAppartment(id, rank, time, ctx)
	}
}

func constructID(cardNode *xmlpath.Node) string {
	adressPath := xmlpath.MustCompile(".//*[@class='hc-address']/p")

	var id string
	n := 1 //xmlpath does not support position() when this was written
	for iter := adressPath.Iter(cardNode); iter.Next(); n++ {
		if n == 2 {
			id = iter.Node().String()
		}
	}

	boligDataPath := xmlpath.MustCompile(".//*[@class='hc-bolig-data']/td")
	for iter := boligDataPath.Iter(cardNode); iter.Next(); {
		id += "|" + iter.Node().String()
	}
	return id
}

func getRank(cardNode *xmlpath.Node) int {
	statusPath := xmlpath.MustCompile(".//*[@class='hc-header']/span")
	rank, ok := statusPath.String(cardNode)
	if !ok {
		log.Printf("Unable to find rank for cardNode %v", cardNode.String())
	}
	rank = strings.TrimPrefix(rank, "DIN PLACERING: ")
	intRank, err := strconv.Atoi(rank)
	if err != nil {
		log.Printf("Unable to convert rank to int %v", rank)
	}
	return intRank
}

func persistAppartment(id string, newRank int, t time.Time, ctx context.Context) {

	key := datastore.NewKey(ctx, appartmentEntityKind, id, 0, nil)

	var r Appartment

	if err := datastore.Get(ctx, key, &r); err != nil {
		if err == datastore.ErrNoSuchEntity {
			r.CurrentRank = newRank
			r.Ranks = []RankPair{{t, newRank}}
			r.ID = id
		} else {
			log.Printf("Problem loading rank with id %v. %v", id, err)
		}
	} else {
		if newRank < r.CurrentRank {
			r.CurrentRank = newRank
		}
		r.Ranks = append(r.Ranks, RankPair{t, newRank})
	}

	if _, err := datastore.Put(ctx, key, &r); err != nil {
		log.Printf("Error persisting appartment %v with key %v: %v", r, key, err)
	}
}
