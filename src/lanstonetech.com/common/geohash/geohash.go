package geohash

import "strings"
import "bytes"

const (
	BASE32               = "0123456789bcdefghjkmnpqrstuvwxyz"
	DIRECTION_TOP        = "top"
	DIRECTION_RIGHT      = "right"
	DIRECTION_BOTTOM     = "bottom"
	DIRECTION_LEFT       = "left"
	BASE32_DIR_TOP       = "p0r21436x8zb9dcf5h7kjnmqesgutwvy"
	BASE32_DIR_RIGHT     = "bc01fg45238967deuvhjyznpkmstqrwx"
	BASE32_DIR_BOTTOM    = "14365h7k9dcfesgujnmqp0r2twvyx8zb"
	BASE32_DIR_LEFT      = "238967debc01fg45kmstqrwxuvhjyznp"
	BASE32_BORDER_RIGHT  = "bcfguvyz"
	BASE32_BORDER_LEFT   = "0145hjnp"
	BASE32_BORDER_TOP    = "prxz"
	BASE32_BORDER_BOTTOM = "028b"
	EVEN                 = "even"
	ODD                  = "odd"
)

var bits = []int{16, 8, 4, 2, 1}
var base32 = []byte(BASE32)
var neighbors = map[string]map[string]string{
	"top": {
		"even": BASE32_DIR_TOP,
		"odd":  BASE32_DIR_RIGHT,
	},
	"right": {
		"even": BASE32_DIR_RIGHT,
		"odd":  BASE32_DIR_TOP,
	},
	"bottom": {
		"even": BASE32_DIR_BOTTOM,
		"odd":  BASE32_DIR_LEFT,
	},
	"left": {
		"even": BASE32_DIR_LEFT,
		"odd":  BASE32_DIR_BOTTOM,
	},
}
var borders = map[string]map[string]string{
	"top": {
		"even": BASE32_BORDER_TOP,
		"odd":  BASE32_BORDER_RIGHT,
	},
	"right": {
		"even": BASE32_BORDER_RIGHT,
		"odd":  BASE32_BORDER_TOP,
	},
	"bottom": {
		"even": BASE32_BORDER_BOTTOM,
		"odd":  BASE32_BORDER_LEFT,
	},
	"left": {
		"even": BASE32_BORDER_LEFT,
		"odd":  BASE32_BORDER_BOTTOM,
	},
}

type LatLng struct {
	Lat float64
	Lng float64
}

type Bound struct {
	Min LatLng
	Mid LatLng
	Max LatLng
}

type Neighbors struct {
	Top         string
	TopRight    string
	Right       string
	BottomRight string
	Bottom      string
	BottomLeft  string
	Left        string
	TopLeft     string
}

// precision说明:
// precision lat bits  lng bits  lat error lng error km error 误差说明
// 1         2         3         ±23       ±23       ±2500    2500公里
// 2         5         5         ±2.8      ±5.6      ±630      630公里
// 3         7         8         ±0.70     ±0.7      ±78        78公里
// 4         10        10        ±0.087    ±0.18     ±20        20公里
// 5         12        13        ±0.022    ±0.022    ±2.4      2.4公里
// 6         15        15        ±0.0027   ±0.0055   ±0.61      60米
// 7         17        18        ±0.00068  ±0.00068  ±0.076      8米
// 8         20        20        ±0.000085 ±0.00017  ±0.019      2米
func Encode(lat, lng float64, precision int) string {
	var geohash bytes.Buffer
	var minLat, maxLat float64 = -90, 90
	var minLng, maxLng float64 = -180, 180
	var mid float64 = 0

	bit := 0
	ch := 0
	length := 0
	isEven := true
	for length < precision {
		if isEven {
			mid = (minLng + maxLng) / 2
			if mid < lng {
				ch |= bits[bit]
				minLng = mid
			} else {
				maxLng = mid
			}
		} else {
			mid = (minLat + maxLat) / 2
			if mid < lat {
				ch |= bits[bit]
				minLat = mid
			} else {
				maxLat = mid
			}
		}

		isEven = !isEven
		if bit < 4 {
			bit++
		} else {
			geohash.WriteByte(base32[ch])
			length++
			bit = 0
			ch = 0
		}
	}
	return geohash.String()
}

func DecodeBounds(geohash string) (LatLng, LatLng) {
	var minLat, maxLat float64 = -90, 90
	var minLng, maxLng float64 = -180, 180
	var mid float64 = 0
	isEven := true
	for _, ch := range strings.Split(geohash, "") {
		bit := bytes.Index(base32, []byte(ch))
		i := uint8(4)
		for {
			mask := (bit >> i) & 1
			if isEven {
				mid = (minLng + maxLng) / 2
				if mask == 1 {
					minLng = mid
				} else {
					maxLng = mid
				}
			} else {
				mid = (minLat + maxLat) / 2
				if mask == 1 {
					minLat = mid
				} else {
					maxLat = mid
				}
			}
			isEven = !isEven

			if i == 0 {
				break
			}
			i--
		}
	}
	return LatLng{minLat, minLng}, LatLng{maxLat, maxLng}
}

func Decode(geohash string) *Bound {
	latlngMin, latlngMax := DecodeBounds(geohash)
	bound := new(Bound)
	bound.Min = latlngMin
	bound.Max = latlngMax
	bound.Mid = LatLng{
		Lat: (latlngMin.Lat + latlngMax.Lat) / 2,
		Lng: (latlngMin.Lng + latlngMax.Lng) / 2,
	}
	return bound
}

func getNeighbor(geohash string, direction string) string {
	length := len(geohash)
	last := geohash[(length - 1):]
	oddEven := ODD
	if (length % 2) == 0 {
		oddEven = EVEN
	}
	border := borders[direction][oddEven]
	base := geohash[0 : length-1]
	if strings.Index(border, last) != -1 && 1 < length {
		base = getNeighbor(base, direction)
	}
	neighbor := neighbors[direction][oddEven]
	return base + string(base32[strings.Index(neighbor, last)])
}

func GetNeighbors(geohash string) Neighbors {
	type result struct {
		direction string
		geohash   string
	}

	worker := func(hash string, direction string, c chan<- result) {
		c <- result{direction, getNeighbor(hash, direction)}
	}

	ch := make(chan result, 8)

	go worker(geohash, DIRECTION_TOP, ch)
	go worker(geohash, DIRECTION_BOTTOM, ch)

	top := <-ch
	bottom := <-ch

	go worker(geohash, DIRECTION_RIGHT, ch)
	go worker(geohash, DIRECTION_LEFT, ch)
	go worker(top.geohash, DIRECTION_RIGHT, ch)
	go worker(top.geohash, DIRECTION_LEFT, ch)
	go worker(bottom.geohash, DIRECTION_RIGHT, ch)
	go worker(bottom.geohash, DIRECTION_LEFT, ch)

	right := <-ch
	left := <-ch
	topRight := <-ch
	topLeft := <-ch
	bottomRight := <-ch
	bottomLeft := <-ch

	return Neighbors{
		Top:         top.geohash,
		TopRight:    topRight.geohash,
		Right:       right.geohash,
		BottomRight: bottomRight.geohash,
		Bottom:      bottom.geohash,
		BottomLeft:  bottomLeft.geohash,
		Left:        left.geohash,
		TopLeft:     topLeft.geohash,
	}
}
