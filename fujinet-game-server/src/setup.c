/**
 * Atari Game Server
 *
 * setup.c - Setup 
 *
 */

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <netinet/in.h>
#include "setup.h"
#include "context.h"

/**
 * Global Server context
 */
extern Context context;

bool setup(int argc, char* argv[])
{
  // Create server socket
  if ((context.listen_fd = socket(AF_INET, SOCK_DGRAM, 0)) < 0)
    {
      perror("socket creation failed");
      return false;
    }

  // Clear address structs
  memset(&context.servaddr, 0, sizeof(context.servaddr));
  memset(&context.clientaddr, 0, sizeof(context.clientaddr));

  // Fill in server information
  context.servaddr.sin_family = AF_INET;
  context.servaddr.sin_addr.s_addr = INADDR_ANY;
  context.servaddr.sin_port = htons(SERVER_PORT);

  // Bind socket with server address
  if (bind(context.listen_fd, (const struct sockaddr *)&context.servaddr, sizeof(context.servaddr)) < 0)
    {
      perror("bind failed");
      return false;
    }
  
  return true;
}
