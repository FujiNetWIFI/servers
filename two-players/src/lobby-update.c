/**
 * @brief a ridiculously simple routine to update the lobby.
 * @author Thomas Cherryhomes
 * @email thom dot cherryhomes at gmail dot com
 * @license gpl v. 3, see LICENSE.md for details.
 */

#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <strings.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netdb.h>

/**
 * @brief the lobby endpoint
 */
const char *lobby_host = "fujinet.online";
const int lobby_port = 8080;

/**
 * @brief the snprintf() template for the HTTP headers
 */
const char *lobby_headers_fmt =
  "POST /server HTTP/1.1\r\n"
  "Host: %s:%u\r\n"
  "User-Agent: two-players/1.0\r\n"
  "Accept: */*\r\n"
  "Content-Type: application/json\r\n"
  "Content-Length: %u\r\n"
  "\r\n";

/**
 * @brief the snprintf() template for the body JSON emitted
 */
const char *lobby_body_fmt =
  "{\r\n"
  "    \"game\": \"%s\",\r\n"
  "    \"gametype\": %u,\r\n"
  "    \"appkey\": %u,\r\n"
  "    \"server\": \"%s\",\r\n"
  "    \"region\": \"%s\",\r\n"
  "    \"serverurl\": \"%s\",\r\n"
  "    \"status\": \"%s\",\r\n"
  "    \"maxplayers\": 2,\r\n"
  "    \"curplayers\": %u,\r\n"
  "    \"clients\": [\r\n"
  "        {\r\n"
  "            \"platform\": \"atari\",\r\n"
  "            \"url\": \"TNFS://apps.irata.online/Atari_8-bit/Games/Reversi.atr\"\r\n"  
  "        }\r\n"
  "    ]\r\n"
  "}\r\n";

/**
 * @brief the snprintf() template for DELETE HTTP headers
 */
const char *lobby_delete_headers_fmt =
  "DELETE /server HTTP/1.1\r\n"
  "Host: %s:%u\r\n"
  "User-Agent: two-players/1.0\r\n"
  "Accept: */*\r\n"
  "Content-Type: application/json\r\n"
  "Content-Length: %u\r\n"
  "\r\n";  

/**
 * @brief the snprintf() template for the body JSON emitted
 */
const char *lobby_delete_body_fmt =
  "{\r\n"
  "    \"serverurl\": \"%s\"\r\n"
  "}\r\n";

/**
 * @brief the buffer used for holding the HTTP request and response.
 */
char lobby_buf[16384];
char lobby_headers_buf[8192];
char lobby_body_buf[8192];

/**
 * @brief send an update message to lobby server
 * @param game Game name
 * @param server_desc Description of the server (blue room, red room, etc.)
 * @param region ISO region name in lowercase. ("us")
 * @param server_url The N: URL to the server ("N:TCP:/...")
 * @param status The status of the server ("online" or "offline")
 * @param curplayers The current # of players (0 or 1)
 */
bool lobby_update(char *game,
		  unsigned char game_type,
		  unsigned char app_key,
		  char *server_desc,
		  char *region,
		  char *server_url,
		  char *status,
		  unsigned char curplayers)
{
  struct hostent *server;
  struct sockaddr_in serv_addr;
  int sockfd;
  size_t bytes, sent, received, total;
  char *success = NULL;
  
  // create socket
  sockfd = socket(AF_INET, SOCK_STREAM, 0);

  if (sockfd < 0)
    {
      perror("lobby_update");
      exit(1);
    }

  // translate host to IP address
  server = gethostbyname(lobby_host);

  if (server == NULL)
    {
      perror("lobby_update");
      exit(1);
    }

  // Fill in address structure
  memset(&serv_addr,0,sizeof(serv_addr));
  serv_addr.sin_family = AF_INET;
  serv_addr.sin_port = htons(lobby_port);
  memcpy(&serv_addr.sin_addr.s_addr,server->h_addr,server->h_length);

  // Connect socket
  if (connect(sockfd, (struct sockaddr *)&serv_addr, sizeof(serv_addr)) < 0)
    {
      perror("lobby_update");
      exit(1);
    }

  // zero buffers
  bzero(lobby_buf,sizeof(lobby_buf));
  bzero(lobby_headers_buf,sizeof(lobby_headers_buf));
  bzero(lobby_body_buf,sizeof(lobby_body_buf));

  // Build body buffer
  snprintf(lobby_body_buf,sizeof(lobby_body_buf),
	   lobby_body_fmt,
	   game,
	   game_type,
	   app_key,
	   server_desc,
	   region,
	   server_url,
	   status,
	   curplayers);
  
  // Build header buffer
  snprintf(lobby_headers_buf,sizeof(lobby_headers_buf),lobby_headers_fmt,lobby_host,lobby_port,strlen(lobby_body_buf));

  // Concatenate into target buffer
  strcpy(lobby_buf,lobby_headers_buf);
  strcat(lobby_buf,lobby_body_buf);

  // echo request
  printf("%s\n",lobby_buf);

  // Send request
  dprintf(sockfd,"%s",lobby_buf);
  
  // Get response
  bzero(lobby_buf,sizeof(lobby_buf));
  read(sockfd,lobby_buf,sizeof(lobby_buf));
  
  close(sockfd);

  printf("Response received: %s\n",lobby_buf);

  return strstr(lobby_buf,"\"success\":true") != NULL;
}

/**
 * @brief issue lobby delete request
 * @param server_url - the server URL to delete
 */
void lobby_delete(char *server_url)
{
  struct hostent *server;
  struct sockaddr_in serv_addr;
  int sockfd;

  // create socket
  sockfd = socket(AF_INET, SOCK_STREAM, 0);

  if (sockfd < 0)
    {
      perror("lobby_update");
      exit(1);
    }

  // translate host to IP address
  server = gethostbyname(lobby_host);

  if (server == NULL)
    {
      perror("lobby_update");
      exit(1);
    }

  // Fill in address structure
  memset(&serv_addr,0,sizeof(serv_addr));
  serv_addr.sin_family = AF_INET;
  serv_addr.sin_port = htons(lobby_port);
  memcpy(&serv_addr.sin_addr.s_addr,server->h_addr,server->h_length);

  // Connect socket
  if (connect(sockfd, (struct sockaddr *)&serv_addr, sizeof(serv_addr)) < 0)
    {
      perror("lobby_update");
      exit(1);
    }

  // zero buffers
  bzero(lobby_buf,sizeof(lobby_buf));
  bzero(lobby_headers_buf,sizeof(lobby_headers_buf));
  bzero(lobby_body_buf,sizeof(lobby_body_buf));

  snprintf(lobby_body_buf,sizeof(lobby_body_buf),
	   lobby_delete_body_fmt,
	   server_url);

  // Build header buffer
  snprintf(lobby_headers_buf,sizeof(lobby_headers_buf),lobby_delete_headers_fmt,lobby_host,lobby_port,strlen(lobby_body_buf));

  // Concatenate into target buffer
  strcpy(lobby_buf,lobby_headers_buf);
  strcat(lobby_buf,lobby_body_buf);

  // echo request
  printf("%s\n",lobby_buf);

  // Send request
  dprintf(sockfd,"%s",lobby_buf);
  
  // Get response
  bzero(lobby_buf,sizeof(lobby_buf));
  read(sockfd,lobby_buf,sizeof(lobby_buf));
  
  close(sockfd);

  printf("Response received: %s\n",lobby_buf);  
}
