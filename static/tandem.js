(function() {

  function TandemSpot(car) {
    var self = this;
    self.parked = ko.observable(car || null);
    self.requests = ko.observableArray([]);

    self.request = function(car) {
      if (!car || self.parked() == car) return;
      if (self.requests.indexOf(car) < 0) {
        if (car.request())
          car.request().requests.remove(car);
        car.request(self);
        self.requests.push(car);
      } else {
        car.request(null);
        self.requests.remove(car);
      }
    }

    self.carHeight = ko.computed(function() {
      var numCars = self.requests().length;
      if (self.parked()) numCars++;
      return 100/numCars + '%';
    });
  }

  function Car(driver) {
    var self = this;
    self.driver = ko.observable(driver || null);
    self.request = ko.observable();
    self.parked = ko.observable();

    self.time = ko.observable();
    self.suggestion = ko.observable();

    self.park = function(spot) {
      if (!self.parked() && spot) {
        self.parked(spot);
        if (spot !== "street") {
          spot.parked(self);
        }
      }
    }

    self.parkRequested = function() {
      if (self.request() && !self.request().parked()) {
        if (self.parked() && self.parked() != "street") {
          self.parked().parked(null);
        }
        self.request().requests.remove(self);
        self.request().parked(self);
        self.parked(self.request());
        self.request(null);
      } else if (self.parked() == "street") {
        self.parked(null);
      } else if (self.parked()) {
        self.parked().parked(null);
        self.parked(null);
      } else {
        self.parked("street");
      }
    }

    self.unpark = function() {
      if (self.parked()) {
        if (self.parked() !== "street") {
          self.parked().parked(null);
        }
        self.parked(null);
      }
    }
  }

  function TandemViewModel() {
    var self = this;
    self.cars = ko.observableArray([]);
    self.spots = ko.observableArray([]);
    self.car = ko.observable();

    self.park = function(street) {
      if (!self.car()) return;
      var spot = "street";
      if (!street) {
        spot = ko.utils.arrayFirst(self.spots(), function(spot) {
          return !spot.parked();
        });
      }
      self.car().park(spot);
    }

    self.unpark = function() {
      if (!self.car() || !self.car().parked()) return;
      if (self.car().parked() !== "street") {
        var spotLoc = self.spots.indexOf(self.car().parked());
        for (var i = spotLoc + 1; i < self.spots().length; i++) {
          var spot = self.spots()[i];
          if (spot.parked()) {
            console.log(self.car().driver(), "blocked");
            return;
          }
        }
      }
      self.car().unpark();
    }

    self.time = ko.observable();
    self.schedule = function() {
      self.car().time(parseInt(self.time()));
      self.time(null);
    }

    self.suggestSpot = function(cars, numSpots) {
      var numCars = cars.length;
      if (numSpots === 0) {
        ko.utils.arrayForEach(cars, function(car) {
          car.suggestion("s");
        });
      } else if (numSpots === 1) {
        if (numCars === 1) {
          cars[0].suggestion("i");
        } else {
          ko.utils.arrayForEach(cars, function(car) {
            car.suggestion("s/i");
          });
        }
      } else {
        var carsBefore = 0;
        var carsAfter = numCars - 1;
        ko.utils.arrayForEach(cars, function(car) {
          if (carsBefore - carsAfter < 0) {
            car.suggestion("s");
          } else {
            car.suggestion("i");
          }
          carsBefore += 1;
          carsAfter -= 1;
        });
      }
    }
    ko.computed(function() {
      self.cars.sort(function(l, r) {
        return l.time() == r.time() ? 0 : (l.time() < r.time() ? -1 : 1)
      });
      var targetCars = [];
      var numSpots = self.spots().filter(function(spot) { return !spot.parked(); }).length;
      ko.utils.arrayForEach(self.cars(), function(car) {
        if (!car.parked()) {
          targetCars.push(car);
        } else {
          car.suggestion("parked");
          if (car.parked() != "street") {
            self.suggestSpot(targetCars, numSpots);
            targetCars = [];
            numSpots = 0;
          }
        }
      });
      self.suggestSpot(targetCars, numSpots);
    });

    self.selectCar = function(car) {
      if (car == self.car())
        self.car(null);
      else
        self.car(car);
    }

    self.save = function() {
      console.log("save");
      var saveData = [];
      saveData.push(self.spots().length);
      saveData.push(self.cars().length);
      ko.utils.arrayForEach(self.cars(), function(car) {
        saveData.push(car.driver());
        saveData.push(car.time() || "");
        if (car.parked() === "street") {
           saveData.push("s");
        } else {
          var spotLoc = self.spots.indexOf(car.parked());
          if (spotLoc !== -1) saveData.push(spotLoc);
          else saveData.push("");
        }
      });
      window.open("https://twitter.com/intent/tweet?screen_name=TandemCommander&text=save "+
                  saveData.join(","));
    }

    $.ajax({
      url: "https://cdn.syndication.twimg.com/widgets/timelines/463924049412771840",
      method: "get",
      dataType: "jsonp"

    }).success(function(data) {
      var parser = document.createElement('div');
      parser.innerHTML = data['body'];
      var tweets = $(parser).find('.e-entry-title');
      var parkingUpdates = {};
      for (var i in tweets) {
        var tweet = tweets[i].innerText;
        console.log("data", tweet);

        /* Check for parking updates */
        var parkStart = tweet.indexOf("park");
        if (parkStart != -1) {
          var parkData = tweet.slice(parkStart+5, tweet.length).split(" ");
          var driver = parkData[0];
          if (!(driver in parkingUpdates)) {
            var spot = parkData[1];
            if (spot !== "street") {
              spot = parseInt(spot);
            }
            parkingUpdates[driver] = spot;
          }
          continue;
        }
        var goneStart = tweet.indexOf("gone");
        if (goneStart != -1) {
          var driver = tweet.slice(goneStart+5, tweet.length);
          if (!(driver in parkingUpdates)) {
            parkingUpdates[driver] = "gone";
          }
          continue;
        }

        /* Check for save data */
        var saveStart = tweet.indexOf("save");
        if (saveStart != -1) {
          var saveDataa = tweet.slice(saveStart+5, tweet.length).split(",");
          var dataIdx = 0;
          var numSpots = parseInt(saveDataa[dataIdx++]);
          console.log("saved spots:", numSpots);
          for (; numSpots>0; numSpots--) {
            self.spots.push(new TandemSpot());
          }
          var numCars = parseInt(saveDataa[dataIdx++]);
          console.log("saved cars:", numCars);
          for (; numCars>0; numCars--) {
            var carData = {
              name: saveDataa[dataIdx++],
              time: saveDataa[dataIdx++],
              parked: saveDataa[dataIdx++]
            }
            var car = new Car(carData.name);
            if (carData.time !== "")
              car.time(parseInt(carData.time));
            if (carData.parked) {
              console.log("park", carData.name, carData.parked);
              if (carData.parked === "s") {
                car.parked("street");
              } else {
                var parkedSpot = self.spots()[parseInt(carData.parked)];
                car.parked(parkedSpot);
                parkedSpot.parked(car);
              }
            }
            console.log("loaded", car.driver());
            self.cars.push(car);
          }
          /* Save found, no more data required */
          break;
        }
      }

      /* Update parking spots */
      for (var driver in parkingUpdates) {
        var spot = parkingUpdates[driver];
        console.log("park", driver, spot);
        var car = ko.utils.arrayFirst(self.cars(), function(car) {
          return car.driver() === driver;
        })
        if (car.parked() && car.parked() !== "street")
          car.parked().parked(null);
        car.parked(null);
        if (spot !== "gone") {
          var newSpot = self.spots()[spot];
          car.parked(newSpot);
          newSpot.parked(car);
        }
      }
    })
  };

  ko.applyBindings(new TandemViewModel());
})();
