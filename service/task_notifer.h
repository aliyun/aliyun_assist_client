#pragma once

#include <functional>
using std::function;

class  TaskNotifer {
public:
	virtual bool init(function<void(const char*)> callback) = 0;
	virtual void unit() = 0;
};