# filesync
sync file betweent server with some client in real time

server config:
{
    "listen":"0.0.0.0:9090",
    "debug":false,  
    "log_file":"server.log",
    "log_file_num":3,
    "moni_dir":[
        {
            "dir":"E:\\MyCode",
            "white_list":[
                "192.168.1.104:9091"
            ]
        }
    ]
}

client config:
{
    "listen":"0.0.0.0:9091",
    "debug":false,
    "log_file":"client.log",
    "log_file_num":3,
    "sync_dir":[
        {
            "server_dir":"E:\\MyCode\\",
            "local_dir":"E:\\MyCodeBak",
            "server_addr":"192.168.1.104:9090"
        }
    ]
}

when client connect with server first(lost heartbeat more than 300 seconds),
server will check all file is exist or not in client,if not exist will send file to client.
then if file create|write|remove|rename ,will send event to client.

