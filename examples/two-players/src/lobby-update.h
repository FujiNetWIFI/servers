/**
 * @brief a ridiculously simple routine to update the lobby.
 * @author Thomas Cherryhomes
 * @email thom dot cherryhomes at gmail dot com
 * @license gpl v. 3, see LICENSE.md for details.
 */

#ifndef LOBBY_UPDATE_H
#define LOBBY_UPDATE_H

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
		  unsigned char curplayers);

/**
 * @brief issue lobby delete request
 * @param server_url - the server URL to delete
 */
void lobby_delete(char *server_url);

#endif /* LOBBY_UPDATE_H */
