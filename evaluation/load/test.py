import time
import math

def main():
   
    while True:

        end = time.time() + 2
        while time.time() < end:
            math.factorial(100)
        time.sleep(8)

if __name__ == '__main__':
    main()
