/**
 * Atari Game Server
 *
 * main.c - Main program
 *
 */

#include <stdio.h>
#include "version.h"
#include "setup.h"
#include "run.h"
#include "done.h"

/**
 * Return code
 */
int ret;

void banner(int argc, char* argv[])
{
  printf(argv[0] " " VERSION "Built " __DATE__" " __TIME__ "\n");
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
