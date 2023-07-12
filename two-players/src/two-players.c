/**
 * @brief A simple two-player TCP server for FujiNet
 * @author many, including Thomas Cherryhomes
 * @email thom dot cherryhomes at gmail dot com
 * @license gpl v. 3, see LICENSE.md for details
 */

#include <stdbool.h>
#include <stdio.h>
#include <netdb.h>
#include <netinet/in.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <unistd.h>
#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <sys/select.h>
#include <sys/time.h>
#include "lobby-update.h"

/**
 * @brief server parameters
 */
#define GAME "Reversi"
#define GAME_TYPE 1
#define APP_KEY 3
#define SERVER_DESC "Main Room"
#define REGION "us"
#define SERVER_URL "TCP://apps.irata.online:1025/"
#define STATUS "online"

/**
 * @brief maximum data size in chars
 */
#define MAX 80

/**
 * @brief needed for the socket functions
 */
#define SA struct sockaddr

/**
 * @brief needed for select()
 */
struct timeval tv;
fd_set rd;

/**
 * @brief This variable gets set to false, when we're asked to exit.
 */
volatile sig_atomic_t running = true;

/**
 * @brief signal handler for SIGTERM and SIGKILL
 * @param the signal # passed in (not used)
 */
void sighandler(int signum)
{
  printf("Signal caught, stopping server.\n");
  running = false;
}

/**
 * @brief wrapper func to send player # update to lobby
 * @param n # of players
 * @return true on success, false on failure
 */
bool update_players(unsigned char n)
{
  return lobby_update(GAME,
		      GAME_TYPE,
		      APP_KEY,
		      SERVER_DESC,
		      REGION,
		      SERVER_URL,
		      STATUS,
		      n);
}

/**
 * @brief reflect TCP data between players
 * @param connfd_1 connected file descriptor for first player socket
 * @param connfd_2 connected file descriptor for second player socket
 */
void reflect(int connfd_1, int connfd_2)
{
  char buff[MAX];
  ssize_t len;
  bool connected = true;
  int r;
  
  printf("reflect()\n");

  // Set up the fd set for select, these contain our two connections.
  FD_ZERO(&rd);
  FD_SET(connfd_1,&rd);
  FD_SET(connfd_2,&rd);

  tv.tv_sec = 0;
  tv.tv_usec = 1000;

  while (connected)
    {
      FD_ZERO(&rd);
      FD_SET(connfd_1,&rd);
      FD_SET(connfd_2,&rd);
      
      tv.tv_sec = 0;
      tv.tv_usec = 1000;

      r = select(connfd_2+1,&rd,NULL,NULL,&tv); // Wait for some activity, for up to a millisecond.

      if (r<0)
	{
	  perror("reflect");
	  close(connfd_1);
	  close(connfd_2);
	  connected = false;
	  running = false;
	  return;
	}
      
      if (FD_ISSET(connfd_1,&rd))
	{
	  len = read(connfd_1, buff, MAX);

	  if (!len) // length = 0 means we disconnected.
	    {
	      connected = false;
	      printf("player 1 disconnected.\n");
	      break;
	    }
	  else
	    write(connfd_2,buff,len);
	}

      if (FD_ISSET(connfd_2,&rd))
	{
	  len = read(connfd_2, buff, MAX);

	  if (!len) // length = 0 means we disconnected.
	    {
	      connected = false;
	      printf("player 2 disconnected.\n");
	      break;
	    }
	  else
	    write(connfd_1,buff,len);
	}
    }

  // Close player connections.
  close(connfd_1);
  close(connfd_2);
}

/**
 * @brief bind, listen, and accept two player connections
 */
int main(int argc, char *argv[])
{
  int sockfd, connfd_1, connfd_2, len, flags, port, r;
  struct sockaddr_in servaddr, cli;
  char c=0;
  
  // Attach sighandler to SIGTERM and SIGKILL Signals ////////////////////////

  signal(SIGTERM, sighandler);
  signal(SIGKILL, sighandler);
  signal(SIGHUP, sighandler);
  signal(SIGINT, sighandler);
  
  // Process port argument ///////////////////////////////////////////////////

  if (argc<2)
    {
      printf("%s <port # 1025-65535>\n",argv[0]);
      return 1;
    }
  else
    {
      port = atoi(argv[1]);

      if (port<1025)
	{
	  printf("%s: invalid port #, use ports 1025-65535\n",argv[0]);
	}
    }
  
  // Listening socket creation and binding ///////////////////////////////////

  // socket create and verification
  sockfd = socket(AF_INET, SOCK_STREAM, 0);
  
  if (sockfd < 0) {
    perror("main() - socket creation failed");
    exit(0);
  }
  else
    printf("Socket successfully created..\n");
  
  bzero(&servaddr, sizeof(servaddr));
  
  // assign IP, PORT
  servaddr.sin_family = AF_INET;
  servaddr.sin_addr.s_addr = htonl(INADDR_ANY);
  servaddr.sin_port = htons(port);
  
  // Binding newly created socket to given IP and verification
  if ((bind(sockfd, (SA*)&servaddr, sizeof(servaddr))) != 0) {
    perror("main() - socket bind failed");
    exit(1);
  }
  else
    printf("Socket successfully bound.\n");

  while (running)
    {
      // WAITING FOR CONNECTION ///////////////////////////////////////////////

      // Update lobby for 0 players
      update_players(0);
      
      // Listen for first player  
      if (listen(sockfd, 1) < 0) {
	perror("main() - Listen failed");
	exit(1);
      }

      len = sizeof(cli);
            
      r = 0;

      printf("Server listening for first player.\n");

      while (r==0)
	{
	  // We need to reset tv and the fd set, each time through the loop.
	  FD_ZERO(&rd);
	  FD_SET(sockfd,&rd);
	  tv.tv_sec = 0;
	  tv.tv_usec = 1000; // 1 millisecond timeout.
	  
	  r = select(sockfd+1,&rd,NULL,NULL,&tv);
	  
	  if (r < 0)
	    {
	      goto bye;
	    }
	  else if (r == 0)
	    {
	      // Go back around again.
	    }
	  else
	    connfd_1 = accept(sockfd, (SA *)&cli, &len);
	}
      
      r = 0;

      // Update lobby for 1 player
      update_players(1);

      printf("Server listening for second player.\n");
      
      while (r==0)
	{
	  FD_ZERO(&rd);
	  FD_SET(sockfd,&rd);
	  tv.tv_sec = 0;
	  tv.tv_usec = 1000;
	  
	  r = select(sockfd+1,&rd,NULL,NULL,&tv);
	  
	  if (r < 0)
	    {
	      goto bye;
	    }
	  else if (r == 0)
	    {
	      // Go back around again.
	    }
	  else
	    connfd_2 = accept(sockfd, (SA *)&cli, &len);
	}

      // Update lobby for 2 players
      update_players(2);
      
      // WE HAVE PLAYERS, PASS TO REFLECT ////////////////////////////////////////////////////

      // First, send player #
      write(connfd_1,&c,1);
      c++;
      write(connfd_2,&c,1);

      // Then pass to reflect.
      reflect(connfd_1,connfd_2);
    }

 bye:
  close(sockfd);
  lobby_delete(SERVER_URL);
}
