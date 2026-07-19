import websocket
import json
import time
import requests
import threading
from inputimeout import inputimeout, TimeoutOccurred

# https://th.carr-designs.com/tables


# Configuration
SERVER_IP = "th.carr-designs.com"
HTTP_PORT = "443"
WS_PORT = "443"
TABLE_ID = "ai2"
PLAYER_NAME = "TestClient"

BASE_URL = f"https://{SERVER_IP}:{HTTP_PORT}"
WEBSOCKET_URL = f"ws://{SERVER_IP}:{WS_PORT}/ws?table={TABLE_ID}&player={PLAYER_NAME}"

# --- Global state to share between threads ---
player_info = {"id": -1, "is_turn": False}
last_state = {}

def get_player_id(players):
    """Find the player ID for our client."""
    for i, player in enumerate(players):
        if player.get('Name') == PLAYER_NAME:
            return i
    return -1

def post_move(move):
    """Sends the player's move to the server via HTTP POST."""
    try:
        url = f"{BASE_URL}/state?table={TABLE_ID}&player={PLAYER_NAME}"
        headers = {'Content-Type': 'application/json'}
        payload = {"move": move}
        print(f"\n>>> Sending move: {move}")
        response = requests.post(url, json=payload, headers=headers, timeout=3)
        response.raise_for_status()
    except requests.exceptions.RequestException as e:
        print(f"Error sending move: {e}")

def display_menu_and_get_input(state):
    """Displays the move menu and captures user input with a timeout."""
    global player_info
    if not player_info['is_turn']:
        return

    valid_moves = state.get('ValidMoves', [])
    move_time = state.get('MoveTime', 30)

    print("\n--- It's your turn! ---")
    for i, move in enumerate(valid_moves):
        print(f"  {i + 1}: {move['n']}")
    print("-------------------------")

    try:
        choice = inputimeout(prompt=f"Choose your move (1-{len(valid_moves)}) [{move_time}s]: ", timeout=move_time)
        
        choice_index = int(choice) - 1
        if 0 <= choice_index < len(valid_moves):
            selected_move = valid_moves[choice_index]['m']
            post_move(selected_move)
        else:
            print("Invalid choice. Letting timer expire.")

    except TimeoutOccurred:
        print("\nNo input received. Letting server timeout the move.")
    except (ValueError, IndexError):
        print("Invalid input. Letting timer expire.")
    finally:
        player_info['is_turn'] = False # Prevent multiple inputs for the same turn

def format_card(card):
    """Formats a card dictionary into a string like 'As' or 'Th'."""
    if not card or 'Rank' not in card or 'Suit' not in card:
        return "??"
    ranks = {1: 'A', 2: '2', 3: '3', 4: '4', 5: '5', 6: '6', 7: '7', 8: '8', 9: '9', 10: 'T', 11: 'J', 12: 'Q', 13: 'K', 14: 'A'}
    suits = {0: '♠', 1: '♥', 2: '♦', 3: '♣'}
    # Ace can be high or low, server seems to use 14 for high
    rank_val = card['Rank']
    if rank_val == 1: rank_val = 14 # Treat Ace as high for display
    return ranks.get(rank_val, '?') + suits.get(card['Suit'], '?')

def print_game_state(state):
    """Prints the game state in a human-readable format."""
    games_played = state.get('gamesPlayed', 0)
    round_num = state.get('Round', 0)
    pot = state.get('Pot', 0)
    community_cards = state.get('CommunityCards', [])
    players = state.get('Players', [])
    last_result = state.get('LastResult', '')

    # A simple way to clear the screen
    print("\033[H\033[J", end="")

    if last_result:
        print(f"Last Result: {last_result}\n")

    round_name = state.get('roundName', f'Round {round_num}')
    print(f"--- Texas Hold'em --- Game #{games_played} --- {round_name} ---")
    community_str = " ".join(format_card(c) for c in community_cards) if community_cards else "Not dealt"
    print(f"Pot: ${pot:<6} | Community Cards: [ {community_str} ]")
    print("-" * 80)
    print(f"  {'Player':<15} {'Status':<12} {'Purse':<10} {'Bet':<10} {'Hand'}")
    print("-" * 80)
    for i, player in enumerate(players):
        name = player.get('Name', 'Unknown')
        status_map = {0: "Waiting", 1: "Playing", 2: "Folded", 3: "All-In", 4: "Left"}
        status = status_map.get(player.get('Status'), "Unknown")
        purse = player.get('Purse', 0)
        bet = player.get('Bet', 0)
        hand = player.get('Hand', [])
        
        # For testing, show all hands
        hand_str = " ".join(format_card(c) for c in hand) if hand else ""

        active_marker = "->" if state.get('ActivePlayer') == i else "  "
        print(f"{active_marker} {name:<15} {status:<12} ${purse:<9} ${bet:<9} {hand_str}")
    print("-" * 80)

def on_message(ws, message):
    """Callback function to handle incoming messages from the WebSocket."""
    global player_info, last_state
    try:
        message_data = json.loads(message)
        state = message_data.get('data', {})
        if not state:
            print(f"Received message without game state data: {message_data}")
            return

        last_state = state
        print_game_state(state)

        if player_info['id'] == -1:
            player_info['id'] = get_player_id(state.get('Players', []))

        active_player_id = state.get('ActivePlayer', -1)
        if player_info['id'] != -1 and active_player_id == player_info['id']:
            player_info['is_turn'] = True
        else:
            player_info['is_turn'] = False

    except json.JSONDecodeError:
        print(f"Received non-JSON message: {message}")

def on_error(ws, error):
    print(f"Error: {error}")

def on_close(ws, close_status_code, close_msg):
    print(f"\n### Connection closed: {close_status_code} - {close_msg} ###")

def on_open(ws):
    print("WebSocket connection opened.")

def run_websocket():
    ws = websocket.WebSocketApp(WEBSOCKET_URL, on_open=on_open, on_message=on_message, on_error=on_error, on_close=on_close)
    ws.run_forever()

if __name__ == "__main__":
    # Initial HTTP request to join the table
    try:
        print(f"Joining table '{TABLE_ID}' as '{PLAYER_NAME}'...")
        requests.get(f"{BASE_URL}/state?table={TABLE_ID}&player={PLAYER_NAME}", timeout=5)
    except requests.exceptions.RequestException as e:
        print(f"Could not connect to server: {e}")
        exit(1)

    # Start WebSocket in a background thread
    ws_thread = threading.Thread(target=run_websocket)
    ws_thread.daemon = True
    ws_thread.start()

    print("Client started. Waiting for game updates...")

    while ws_thread.is_alive():
        if player_info['is_turn']:
            display_menu_and_get_input(last_state)
        time.sleep(0.5) # Main thread polls to check if it's our turn

    print("Exiting.")
