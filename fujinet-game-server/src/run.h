/**
 * Atari Game Server
 *
 * run.h - Program main loop 
 *
 */

#ifndef RUN_H
#define RUN_H

#include <stdbool.h>

/**
 * @brief Server Main loop
 * @param argc CLI argument count 
 * @param argv CLI argument array
 * @return TRUE if run completed, FALSE if not.
 */
bool run(int argc, char* argv[]);

#endif /* RUN_H */
