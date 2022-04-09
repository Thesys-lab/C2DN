#pragma once

namespace Utils {
long convert_size(std::string str_size) {
  if (std::tolower(str_size.back()) == 'm') {
    str_size.pop_back();
    return std::stol(str_size) * MB;
  } else if (std::tolower(str_size.back()) == 'g') {
    str_size.pop_back();
    return std::stol(str_size) * GB;
  } else if (std::tolower(str_size.back()) == 't') {
    str_size.pop_back();
    return std::stol(str_size) * TB;
  } else {
    return std::stol(str_size);
  }
}

}