#include <cstdio>
#include <iostream>
#include <memory>
#include <string>
using namespace std;

class Enemy {
public:
  string type;
  int attackPower;
  int health;

  Enemy(string startingType, int startingAttackpower, int startingHealth) {
    type = startingType;
    attackPower = startingAttackpower;
    health = startingHealth;
  }

  bool attack() {
    printf("%s takes a swing and deals a %d", type.c_str(), attackPower);
    return true;
  }
};
class Player {
public:
  string type;
  int attackPower;
  int health;

  Player(string startingType, int startingAttackpower, int startingHealth) {
    type = startingType;
    attackPower = startingAttackpower;
    health = startingHealth;
  }

  bool attack() {
    printf("%s takes a swing and deals a %d", type.c_str(), attackPower);
    return true;
  }
};

int main() {
  unique_ptr<Enemy> e = make_unique<Enemy>("silverman", 12, 100);
  printf("a %s appeared with %d and %d health points", e->type.c_str(),
         e->attackPower, e->health);

  return 0;
}
