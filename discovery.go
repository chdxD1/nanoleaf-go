package nanoleaf

import (
	"fmt"
	"time"

	"github.com/hashicorp/mdns"
)

const NANOLEAF_API_URL = "/api/v1"
const NANOLEAF_MDNS_NAME = "_nanoleafapi._tcp"

func discoverNanoleafs(timeout time.Duration) ([]string, error) {
	entriesCh := make(chan *mdns.ServiceEntry)
	params := mdns.DefaultParams(NANOLEAF_MDNS_NAME)
	params.Entries = entriesCh
	params.Timeout = timeout
	err := mdns.Query(params)
	close(entriesCh)
	if err != nil {
		return nil, err
	}

	urls := []string{}

	for entry := range entriesCh {
		urls = append(urls, fmt.Sprintf("http://%s:%d%s", entry.AddrV4, entry.Port, NANOLEAF_API_URL))
	}
	return urls, nil
}

func DiscoverNanoleafs(timeout time.Duration) (nanoleafs []*Nanoleaf, err error) {
	nanoleafUrls, err := discoverNanoleafs(timeout)
	if err != nil {
		return nil, err
	}
	for _, url := range nanoleafUrls {
		nanoleafs = append(nanoleafs, NewNanoleaf(url))
	}
	return nanoleafs, nil
}
