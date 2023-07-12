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
 * @brief the snprintf() template for the header + JSON emitted
 */
const char *lobby_template =
  "POST /server HTTP/1.1\r\n"
  "Host: %s:%u\r\n"
  "User-Agent: two-players/1.0\r\n"
  "Accept: */*\r\n"
  "Content-type: application/json\r\n\r\n"
  "{"
  "    \"game\": \"%s\",\r\n"
  "    \"gametype\": %u,\r\n"
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
 * @brief the buffer used for holding the HTTP response.
 */
char lobby_response_buf[2048];

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
		  char *server_desc,
		  char *region,
		  char *server_url,
		  char *status,
		  unsigned char curplayers)
{
  struct hostent *server;
  struct sockaddr_in serv_addr;
  int sockfd;
  size_t bytes, received, total;
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

  // echo request
  printf("Request: ");
  printf(lobby_template,
	 lobby_host,
	 lobby_port,
	 game,
	 game_type,
	 server_desc,
	 region,
	 server_url,
	 status,
	 curplayers);
  
  // Send request
  dprintf(sockfd,
	  lobby_template,
	  lobby_host,
	  lobby_port,
	  game,
	  game_type,
	  server_desc,
	  region,
	  server_url,
	  status,
	  curplayers);

  // Get response
  bzero(lobby_response_buf,sizeof(lobby_response_buf));

  total = sizeof(lobby_response_buf) - 1;
  received = 0;

  do
    {
      bytes = read(sockfd, lobby_response_buf+received,total-received);

      if (bytes < 0)
	{
	  perror("lobby_update");
	  exit(1);
	}
      if (bytes == 0)
	break;

      received += bytes;
    } while (received < total);

  if (received == total)
    {
      printf("Response too large\r\n");
      exit(1);
    }

  close(sockfd);

  printf("Response received: %s\n",lobby_response_buf);

  return strstr(lobby_response_buf,"\"success\":true") != NULL;
}
