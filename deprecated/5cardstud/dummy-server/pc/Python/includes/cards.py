import os
import sys
import random

# Local imports
my_modules_path = os.getcwd()+"/includes"
if sys.path[0] != my_modules_path:
    sys.path.insert(0, my_modules_path)
    
IMAGE_PATH_CARDS = 'images/cards/'
CARDBACK_FILENAME = "cardback1.png"

class Cards:
    def __init__(self, decks):
        self.cards = {}
        self.suites = {}
        self.decks = decks
        self.create_cards()
        self.face_down = IMAGE_PATH_CARDS + CARDBACK_FILENAME
    
    def create_cards(self):
        
        self.cards = {}
        self.suites = { "C": "clubs", "D":"diamonds", "H":"hearts", "S":"spades" }
        for s in ['C','D','H','S']:
            for c in range(2,9+1):
                card = str(c) + s
                filename = IMAGE_PATH_CARDS + str(c) + "_of_" + self.suites[s] + ".png"
                self.cards[card] = filename
                
            for c in ['jack','queen','king','ace']:
                card = c[0].upper() + s
                filename = IMAGE_PATH_CARDS + str(c) + "_of_" + self.suites[s] + ".png"
                self.cards[card] = filename
        
        
                
    def select_card(self, card):
        if card == '??':
            return self.face_down
        else:
            return self.cards[card]
        
    
    def get_filename(card):
        return self.cards[card]

    def create_card_pool(self, decks):
        
        self.decks = decks
        
        self.shoe = int(decks * 0.18)
        
        self.pool = []
        for i in range(decks):
            for key in self.cards.keys():
                self.pool.append(key)
        
        # shuffle
        random.shuffle(self.pool)
        
        return
    
    def get_card(self):
        if len(self.pool) <= self.shoe:
            create_card_pool(self.decks)
            
        card = self.pool[0]
        self.pool.remove(card)
        return card
    
    def top_of_pool(self,card):
        self.pool.insert(0, card)

    def bottom_of_pool(self,card):
        self.pool.append(card)
        
    
    

    