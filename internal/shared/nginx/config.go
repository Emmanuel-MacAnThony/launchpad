package nginx

type Config struct {
	ServiceID string `json:"service_id"`
	Domain    string `json:"domain"`
	Host      string `json:"host"`
	BluePort  int    `json:"blue_port"`
	GreenPort int    `json:"green_port"`
}
