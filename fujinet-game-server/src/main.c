/**
 * Atari Game Server
 *
 * main.c - Main program
 *
 */

#include <stdio.h>
#include "setup.h"
#include "run.h"
#include "done.h"

/**
 * Return code
 */
int ret;

void banner(void)
{
  printf("atari-game-server " __VERSION__ "Built " __DATE__ __TIME__);
}

int main(int argc, char* argv[])
{
  banner(argc, argv);
  
  if (!setup(argc, argv))
    return ret;

  if (!run(argc, argv))
    return ret;

  if (!done(argc, argv))
    return ret;

  return 0;
}
