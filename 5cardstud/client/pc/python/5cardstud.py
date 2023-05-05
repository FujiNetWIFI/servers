# Standard imports
import time
import sys
import os
os.environ['PYGAME_HIDE_SUPPORT_PROMPT'] = "hide"
import pygame

# Local imports
my_modules_path = os.getcwd()+"/includes"
if sys.path[0] != my_modules_path:
    sys.path.insert(0, my_modules_path)
    
from common import *

from json_handler import *
from cards import Cards
from drawing import *
from event_handler import *
from button_handler import *

global chips_start_pos

class Poker(object):
    global chips_start_pos
    
    def clean_screen(self):
        common_vars.screen.fill(GAME_BOARD_COLOR)
        common_vars.screen.blit(image_db.get_image(yellow_box), (common_vars.yellow_box_x, common_vars.yellow_box_y)) 
        common_vars.screen.blit(image_db.get_image(IMAGE_PATH + "fujinet_banner.png"), (10, 10))

    # Initialize pygame hooks
    pygame.init()
    pygame.display.set_caption('Fujinet 5 Card Stud')
    pygame.font.init()
    clock = pygame.time.Clock()

    # Instantiate the common variable singleton objects
    common_vars   = CommonVariables.get_instance()
    image_db      = ImageDB.get_instance()

    # Populate the needed common variables with initial values
    common_vars.done        = False
    common_vars.screen      = pygame.display.set_mode(GAME_BOARD_SIZE)
    common_vars.player_cash = DEFAULT_PLAYER_BALANCE
    common_vars.game_rounds = 0
    common_vars.pause_time  = 0
    common_vars.dealer_last_hand = 0
    
    common_vars.player_hand = []
    common_vars.name_pos    = []
    common_vars.player_purse= []
    
    #*****************************************************************************
    # MAX PLAYERS
    #*****************************************************************************
    for i in range(MAX_PLAYERS):
        common_vars.player_hand.append([])
        common_vars.name_pos.append([])
        common_vars.player_purse.append([])
    
    url = 'https://5card.carr-designs.com'

    common_vars.server = json_handler(url)

    if not common_vars.server.connected:
        print(f"{url} is down")
        exit(-1)
    
    common_vars.server.set_table("NORM")
    common_vars.server.set_players(8)
    
    common_vars.button_image_width  = image_db.get_image(IMAGE_PATH_BUTTONS + INACTIVE_BUTTON_FILENAME).get_width()
    common_vars.button_image_height = image_db.get_image(IMAGE_PATH_BUTTONS + INACTIVE_BUTTON_FILENAME).get_height()
    
    common_vars.chips_image_width   = image_db.get_image(IMAGE_PATH_CHIPS   + CHIP_5_FILENAME_ON).get_width()
    common_vars.chips_image_height  = image_db.get_image(IMAGE_PATH_CHIPS   + CHIP_5_FILENAME_ON).get_height()

    common_vars.back_card           = image_db.get_image(IMAGE_PATH_CARDS   + CARDBACK_FILENAME)
    
    common_vars.card_width          = image_db.get_image(IMAGE_PATH_CARDS   + CARDBACK_FILENAME).get_width() 
    common_vars.card_height         = image_db.get_image(IMAGE_PATH_CARDS   + CARDBACK_FILENAME).get_height() 
    
    common_vars.text_font           = pygame.font.SysFont('Arial', 18)  # bold=True
    value_of_players_hand_font      = pygame.font.SysFont('Arial', 16)

    # Plot the base table
    # TODO: Can handle scaling much better to be prepared for other board sizes.
    
    yellow_box = IMAGE_PATH + 'yellow_box_179_120.png'
    
    common_vars.yellow_box_width  = image_db.get_image(yellow_box).get_width()
    common_vars.yellow_box_height = image_db.get_image(yellow_box).get_height()
    common_vars.yellow_box_x = GAME_BOARD_X_SIZE / 2 - common_vars.yellow_box_width / 2
    common_vars.yellow_box_y = GAME_BOARD_Y_SIZE / 2 - common_vars.yellow_box_height / 2
    
    message1 = common_vars.text_font.render('$ 00000 ', False, YELLOW_COLOR)
    common_vars.font_height = message1.get_rect()[3]
    max_bet_width = message1.get_rect()[2]
    current_state = 0

    screen_size_x, screen_size_y = GAME_BOARD_SIZE
    #chips_start_pos   = (screen_size_x - shove, 360)

    common_vars.screen.fill(GAME_BOARD_COLOR)
    common_vars.screen.blit(image_db.get_image(yellow_box), (common_vars.yellow_box_x, common_vars.yellow_box_y)) 
    common_vars.screen.blit(image_db.get_image(IMAGE_PATH + "fujinet_banner.png"), (10, 10))
    
    sound_db = SoundDB.get_instance()
    card_sound = sound_db.get_sound(SOUND_PATH + 'cardslide.wav')

    common_vars.player_bet = []
    common_vars.default_purse_pos = []
    common_vars.default_name_pos = []
    common_vars.name = []
    common_vars.new_card_added = []
    common_vars.player_hand = []
    


    for i in range(8):
        
        x_pos = PLAYER_CARD_START_POS[i][0]
        
        common_vars.player_hand.append([])
        
        common_vars.default_name_pos.append( [x_pos, PLAYER_CARD_START_POS[i][1]-common_vars.font_height] )
        
        common_vars.player_bet.append(0)
        
        common_vars.new_card_added.append(False)
        
        if i > 4:
            common_vars.default_purse_pos.append(  [x_pos - max_bet_width,
                                          PLAYER_CARD_START_POS[i][1] + common_vars.card_height / 2 - common_vars.font_height / 2])
        else:
            common_vars.default_purse_pos.append(  [x_pos + GAP_BETWEEN_CARDS_HORIZ*5 + common_vars.card_width,
                                          PLAYER_CARD_START_POS[i][1] + common_vars.card_height / 2 - common_vars.font_height / 2])

    common_vars.default_purse_pos[0] = [  PLAYER_CARD_START_POS[0][0] + max_bet_width,
                                          PLAYER_CARD_START_POS[0][1] + common_vars.card_height ]

    common_vars.default_purse_pos[4] = [  PLAYER_CARD_START_POS[4][0] + max_bet_width,
                                          PLAYER_CARD_START_POS[4][1] + common_vars.card_height ]

    common_vars.cards = Cards(NUM_OF_DECKS)

    # WHERE THE PLAYERS WILL BE SEATED BASED ON THE NUMBER OF PLAYERS
    
    #
    #      4
    #  3       5
    #  2       6
    #  1       7
    #      0
    #
    common_vars.player_pos_by_count = []
    common_vars.player_pos_by_count.append([0])
    common_vars.player_pos_by_count.append([0,4])
    common_vars.player_pos_by_count.append([0,2,6])
    common_vars.player_pos_by_count.append([0,2,4,6])
    common_vars.player_pos_by_count.append([0,2,3,4,6])
    common_vars.player_pos_by_count.append([0,2,3,4,5,6])
    common_vars.player_pos_by_count.append([0,1,2,3,4,5,6])
    common_vars.player_pos_by_count.append([0,1,2,3,4,5,6,7])
    
    # **********************************************
    # ALL STATIC VARIABLES HAVE BEEN CALCULATED
    # **********************************************

    common_vars.play_again = True
    while common_vars.play_again and not common_vars.done:
        
        # start of new game    
        common_vars.num_players = common_vars.server.get_number_of_players()
        me = 0
        
        common_vars.player_card_start_pos = []
        common_vars.player_name_start_pos = []
        common_vars.name_pos = []
        common_vars.purse_pos = []
        
        seating = common_vars.player_pos_by_count[common_vars.num_players-1]

        for player_num in range(common_vars.num_players):
            seat = seating[player_num]

            common_vars.player_card_start_pos.append(PLAYER_CARD_START_POS[seat])
            common_vars.player_name_start_pos.append(PLAYER_NAME_START_POS[seat])
            common_vars.name_pos.append(common_vars.default_name_pos[seat])
            common_vars.purse_pos.append(common_vars.default_purse_pos[seat])
        
        # no players have folded yet  
        common_vars.player_fold=[]
        for i in range(common_vars.num_players):
            common_vars.player_fold.append(False)
            common_vars.new_card_added[i] = False;
            
        common_vars.player_purse=[]
        for i in range(common_vars.num_players):
            common_vars.name.append(common_vars.server.get_name(i))
            common_vars.player_purse.append(common_vars.server.get_purse(i))
        
        common_vars.dealing = True
        common_vars.hand_in_progress = True
        first_time = True
        
            
        # Main game loop
        common_vars.player_bets = []
                    
        common_vars.new_card_data = True
        common_vars.new_chip_data = False 
        
        common_vars.current_player_num    = 0

        common_vars.get_new_data          = True
        common_vars.dealing               = True
        common_vars.dealing_card_num      = 0
        common_vars.last_dealing_card_num = 0
        
        common_vars.game_in_progress      = True
        common_vars.get_status			  = True
        common_vars.round                 = -1
        update_screen 					  = 1
   
   
        while common_vars.hand_in_progress and not common_vars.done:
            event_handler()
            data_change = common_vars.server.refresh_data()
            
            if not common_vars.server.connected:
                print(f"{url} is down.")
                break
            
            if (not data_change) and (not first_time):
                time.sleep(0.5)
                continue
            
            active_player = common_vars.server.get_active_player()
        
            
            current_round = common_vars.server.get_round()
            
            if current_round == 5:
                common_vars.hand_in_progress = False
            
            buttons = common_vars.server.get_valid_buttons()

            common_vars.screen.fill(GAME_BOARD_COLOR)
            common_vars.screen.blit(image_db.get_image(yellow_box), (common_vars.yellow_box_x, common_vars.yellow_box_y)) 
            common_vars.screen.blit(image_db.get_image(IMAGE_PATH + "fujinet_banner.png"), (10, 10))                                       
            
            erase_buttons()
            if active_player == me:
                draw_buttons(common_vars.screen, buttons)
            
            last_buttons = buttons
            common_vars.round = current_round
            
            for i in range(common_vars.num_players):
                common_vars.player_fold[i] = common_vars.server.get_fold(i)
                common_vars.player_bet[i]  = common_vars.server.get_bet(i)
                
                if common_vars.server.get_playing(i):
                    common_vars.current_player_num = i
                
                hand = common_vars.server.get_hand(i)
                num_cards = int(len(hand)/2)
                common_vars.player_hand[i] = []
                for c in range(num_cards):
                    card = hand[c*2:c*2+2]
                    common_vars.player_hand[i].append(card)
            
            # Plot the players current credits and number of played rounds.
            if active_player != -1:
                x_pos, y_pos = STATUS_START_POS
                message1 = common_vars.text_font.render('Round: {0}'.format(
                    common_vars.round), False, YELLOW_COLOR)  
                common_vars.screen.blit(message1, (x_pos, y_pos))
                                
            max_cards = 0
            for player_num in range(common_vars.num_players):
                m = len(common_vars.player_hand[player_num])
                if m > max_cards:
                    max_cards = m
                draw_name(common_vars.screen, player_num)
            
            slow_draw = first_time
            first_time = False
            
            for dealing_card_num in range(max_cards):
                for player_num in range(common_vars.num_players):
                    card_dealt = common_vars.player_hand[player_num]                                                   
                    
                    drawn = draw_cards(common_vars.screen,
                                   common_vars.player_card_start_pos[player_num],
                                   card_dealt,
                                   dealing_card_num+1)
                    
                    if slow_draw and drawn:
                        pygame.display.flip()
                        card_sound.play()
                        time.sleep(0.3)
                    
                    event_handler()                 
            
            
                for player_num in range(common_vars.num_players):                            
                    draw_purse(common_vars.screen, player_num)
                            
                draw_pot(common_vars.screen)
                
            # Update the content of the display
            pygame.display.flip()            

        # end while hand in progress

        print("Hand ended")

        x_pos, y_pos = STATUS_START_POS
        message1 = common_vars.text_font.render('{0}'.format(
            common_vars.server.get_last_result()), False, RED_COLOR)  
        common_vars.screen.blit(message1, (x_pos, y_pos))
        pygame.display.flip()
        
        if not common_vars.done:
            for i in range(100):
                time.sleep(0.1)
                event_handler()
            print(f"New game")
    #end while play again
    print("Done.")
    pygame.quit()

if __name__ == '__main__':
    MY_GAME = Poker()
    