package nginx

const confTemplate = `upstream {{.ServiceID}}_blue {
    server {{.Host}}:{{.BluePort}};
}

upstream {{.ServiceID}}_green {
    server {{.Host}}:{{.GreenPort}};
}

server {
    listen 80;
    server_name {{.Domain}};

    location / {
        proxy_pass http://{{.ServiceID}}_{{.ActiveSlot}};
    }
}
`

type templateData struct {
	Config
	ActiveSlot string
}
